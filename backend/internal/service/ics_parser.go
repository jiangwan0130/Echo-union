package service

import (
	"fmt"
	"io"
	"math"
	"net/http"
	"sort"
	"strings"
	"time"

	ics "github.com/arran4/golang-ical"

	"echo-union/backend/internal/model"
)

// ── ICS 解析器 ──────────────────────────────────────────────
//
// 职责：将标准 iCalendar (RFC 5545) 内容解析为 CourseSchedule 列表。
//
// 设计决策：
//   - DTSTART/DTEND 确定星期几与时间
//   - RRULE 确定重复模式 → 映射到学期周次
//   - 无 RRULE 的单次事件仅填对应周次
//   - 合并同 name+day+time 不同周次的事件（ICS 可能以多个单次事件表示同一课程）
//   - week_type 由 weeks 数组自动推导
// ─────────────────────────────────────────────────────────────

const (
	icsMaxFileSize   = 5 * 1024 * 1024 // 5MB
	icsFetchTimeout  = 30 * time.Second
	shanghaiTimezone = "Asia/Shanghai"
)

// parsedCourseEvent ICS 解析中间结构
type parsedCourseEvent struct {
	Name      string
	DayOfWeek int // 1=Monday … 7=Sunday
	StartTime string
	EndTime   string
	Weeks     []int
}

// FetchICSContent 从 URL 获取 ICS 内容
func FetchICSContent(rawURL string) (io.ReadCloser, error) {
	// webcal:// → https://
	u := rawURL
	if strings.HasPrefix(u, "webcal://") {
		u = "https://" + strings.TrimPrefix(u, "webcal://")
	}

	client := &http.Client{Timeout: icsFetchTimeout}
	resp, err := client.Get(u)
	if err != nil {
		return nil, fmt.Errorf("获取 ICS 失败: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("获取 ICS 失败: HTTP %d", resp.StatusCode)
	}
	// 限制响应体大小，防止恶意 URL 返回超大内容导致 OOM
	return struct {
		io.Reader
		io.Closer
	}{
		Reader: io.LimitReader(resp.Body, icsMaxFileSize),
		Closer: resp.Body,
	}, nil
}

// ParseICS 解析 ICS 内容并转为 CourseSchedule 列表
//
// 参数：
//   - reader: ICS 数据流
//   - userID, semesterID: 归属信息
//   - semesterStart: 学期起始日期（用于推算周次）
//   - totalWeeks: 学期总周数（默认 25）
func ParseICS(reader io.Reader, userID, semesterID string, semesterStart, semesterEnd time.Time) ([]model.CourseSchedule, error) {
	cal, err := ics.ParseCalendar(reader)
	if err != nil {
		return nil, fmt.Errorf("ICS 格式解析失败: %w", err)
	}

	loc, _ := time.LoadLocation(shanghaiTimezone)
	totalWeeks := calculateTotalWeeks(semesterStart, semesterEnd)

	// 阶段 1: 解析所有 VEVENT
	var events []parsedCourseEvent
	for _, comp := range cal.Events() {
		evt, ok := parseVEvent(comp, semesterStart, totalWeeks, loc)
		if !ok {
			continue
		}
		events = append(events, evt)
	}

	// 阶段 2: 合并同课程（name+day+startTime+endTime 相同）的周次
	merged := mergeEvents(events)

	// 阶段 3: 转为 model.CourseSchedule
	result := make([]model.CourseSchedule, 0, len(merged))
	for _, evt := range merged {
		sort.Ints(evt.Weeks)
		weekType := deriveWeekType(evt.Weeks)
		result = append(result, model.CourseSchedule{
			UserID:     userID,
			SemesterID: semesterID,
			CourseName: evt.Name,
			DayOfWeek:  evt.DayOfWeek,
			StartTime:  evt.StartTime,
			EndTime:    evt.EndTime,
			WeekType:   weekType,
			Weeks:      model.IntArray(evt.Weeks),
			Source:     "ics",
		})
	}
	return result, nil
}

// parseVEvent 解析单个 VEVENT 组件
func parseVEvent(evt *ics.VEvent, semesterStart time.Time, totalWeeks int, loc *time.Location) (parsedCourseEvent, bool) {
	summary := evt.GetProperty(ics.ComponentPropertySummary)
	if summary == nil || strings.TrimSpace(summary.Value) == "" {
		return parsedCourseEvent{}, false
	}
	name := strings.TrimSpace(summary.Value)

	dtStart, err := parseICSDateTime(evt, ics.ComponentPropertyDtStart, loc)
	if err != nil {
		return parsedCourseEvent{}, false
	}
	dtEnd, err := parseICSDateTime(evt, ics.ComponentPropertyDtEnd, loc)
	if err != nil {
		// 若无 DTEND，尝试用 DURATION
		durProp := evt.GetProperty(ics.ComponentPropertyDuration)
		if durProp != nil {
			// 简化处理：默认 2 小时
			dtEnd = dtStart.Add(2 * time.Hour)
		} else {
			return parsedCourseEvent{}, false
		}
	}

	dayOfWeek := goWeekdayToISO(dtStart.Weekday())
	startTime := dtStart.Format("15:04")
	endTime := dtEnd.Format("15:04")

	// 计算周次
	weeks := computeWeeks(evt, dtStart, semesterStart, totalWeeks, loc)
	if len(weeks) == 0 {
		return parsedCourseEvent{}, false
	}

	return parsedCourseEvent{
		Name:      name,
		DayOfWeek: dayOfWeek,
		StartTime: startTime,
		EndTime:   endTime,
		Weeks:     weeks,
	}, true
}

// computeWeeks 根据 RRULE / EXDATE / 单次事件计算周次列表
func computeWeeks(evt *ics.VEvent, dtStart, semesterStart time.Time, totalWeeks int, loc *time.Location) []int {
	rruleProp := evt.GetProperty(ics.ComponentPropertyRrule)
	if rruleProp == nil {
		// 单次事件 → 仅当前周
		wk := dateToWeekNumber(dtStart, semesterStart)
		if wk >= 1 && wk <= totalWeeks {
			return []int{wk}
		}
		return nil
	}

	// 解析 RRULE
	rule := parseRRule(rruleProp.Value)
	if rule.freq != "WEEKLY" {
		// 非周重复 → 视为每周
		wk := dateToWeekNumber(dtStart, semesterStart)
		if wk >= 1 && wk <= totalWeeks {
			return []int{wk}
		}
		return nil
	}

	// 解析 EXDATE
	exDates := parseExDates(evt, loc)

	// 生成所有重复日期
	interval := rule.interval
	if interval < 1 {
		interval = 1
	}

	var weeks []int
	weekSet := make(map[int]bool)

	current := dtStart
	maxDate := semesterStart.AddDate(0, 0, totalWeeks*7)
	if !rule.until.IsZero() && rule.until.Before(maxDate) {
		maxDate = rule.until
	}

	count := 0
	for {
		if !rule.until.IsZero() && current.After(rule.until) {
			break
		}
		if rule.count > 0 && count >= rule.count {
			break
		}
		if current.After(maxDate) {
			break
		}

		wk := dateToWeekNumber(current, semesterStart)
		if wk >= 1 && wk <= totalWeeks {
			dateStr := current.Format("20060102")
			if !exDates[dateStr] && !weekSet[wk] {
				weekSet[wk] = true
				weeks = append(weeks, wk)
			}
		}

		count++
		current = current.AddDate(0, 0, 7*interval)
	}

	return weeks
}

// rruleParams RRULE 解析结果
type rruleParams struct {
	freq     string
	interval int
	count    int
	until    time.Time
}

// parseRRule 解析 RRULE 字符串（如 FREQ=WEEKLY;COUNT=16;INTERVAL=1）
func parseRRule(value string) rruleParams {
	r := rruleParams{interval: 1}
	for _, part := range strings.Split(value, ";") {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		switch strings.ToUpper(kv[0]) {
		case "FREQ":
			r.freq = strings.ToUpper(kv[1])
		case "INTERVAL":
			fmt.Sscanf(kv[1], "%d", &r.interval)
		case "COUNT":
			fmt.Sscanf(kv[1], "%d", &r.count)
		case "UNTIL":
			t, err := time.Parse("20060102T150405Z", kv[1])
			if err != nil {
				t, _ = time.Parse("20060102", kv[1])
			}
			r.until = t
		}
	}
	return r
}

// parseExDates 解析事件中所有 EXDATE
func parseExDates(evt *ics.VEvent, loc *time.Location) map[string]bool {
	exDates := make(map[string]bool)
	for _, prop := range evt.Properties {
		if prop.IANAToken == string(ics.ComponentPropertyExdate) {
			t, err := time.Parse("20060102T150405Z", prop.Value)
			if err != nil {
				t, err = time.Parse("20060102T150405", prop.Value)
				if err != nil {
					t, err = time.Parse("20060102", prop.Value)
				}
			}
			if err == nil {
				exDates[t.In(loc).Format("20060102")] = true
			}
		}
	}
	return exDates
}

// mergeEvents 合并相同课程事件的周次
func mergeEvents(events []parsedCourseEvent) []parsedCourseEvent {
	type key struct {
		Name      string
		DayOfWeek int
		StartTime string
		EndTime   string
	}
	merged := make(map[key]*parsedCourseEvent)
	order := []key{}

	for _, e := range events {
		k := key{Name: e.Name, DayOfWeek: e.DayOfWeek, StartTime: e.StartTime, EndTime: e.EndTime}
		if existing, ok := merged[k]; ok {
			weekSet := make(map[int]bool)
			for _, w := range existing.Weeks {
				weekSet[w] = true
			}
			for _, w := range e.Weeks {
				if !weekSet[w] {
					existing.Weeks = append(existing.Weeks, w)
				}
			}
		} else {
			cp := e
			merged[k] = &cp
			order = append(order, k)
		}
	}

	result := make([]parsedCourseEvent, 0, len(merged))
	for _, k := range order {
		result = append(result, *merged[k])
	}
	return result
}

// ── 辅助函数 ──

// goWeekdayToISO 将 Go 的 time.Weekday (0=Sunday) 转为 ISO 8601 (1=Monday … 7=Sunday)
func goWeekdayToISO(wd time.Weekday) int {
	if wd == time.Sunday {
		return 7
	}
	return int(wd)
}

// dateToWeekNumber 计算日期相对学期起始的周次（1-based）
func dateToWeekNumber(date, semesterStart time.Time) int {
	d := date.Truncate(24 * time.Hour)
	s := semesterStart.Truncate(24 * time.Hour)
	days := int(d.Sub(s).Hours() / 24)
	if days < 0 {
		return 0
	}
	return days/7 + 1
}

// calculateTotalWeeks 计算学期总周数
func calculateTotalWeeks(start, end time.Time) int {
	days := int(end.Sub(start).Hours() / 24)
	weeks := int(math.Ceil(float64(days) / 7.0))
	if weeks < 1 {
		weeks = 25 // 默认上限
	}
	return weeks
}

// deriveWeekType 根据 weeks 数组推导 week_type 冗余字段
func deriveWeekType(weeks []int) string {
	if len(weeks) == 0 {
		return "all"
	}
	allOdd, allEven := true, true
	for _, w := range weeks {
		if w%2 == 0 {
			allOdd = false
		} else {
			allEven = false
		}
	}
	if allOdd {
		return "odd"
	}
	if allEven {
		return "even"
	}
	return "all"
}

// parseICSDateTime 从 VEVENT 中解析日期时间属性
func parseICSDateTime(evt *ics.VEvent, propName ics.ComponentProperty, loc *time.Location) (time.Time, error) {
	prop := evt.GetProperty(propName)
	if prop == nil {
		return time.Time{}, fmt.Errorf("missing property %s", propName)
	}
	val := prop.Value

	// 尝试多种 ICS 日期格式
	formats := []string{
		"20060102T150405Z",
		"20060102T150405",
		"20060102",
	}

	// 检查 TZID 参数
	tzid := ""
	for k, v := range prop.ICalParameters {
		if strings.ToUpper(k) == "TZID" && len(v) > 0 {
			tzid = v[0]
		}
	}

	for _, fmt := range formats {
		if t, err := time.Parse(fmt, val); err == nil {
			if strings.HasSuffix(fmt, "Z") {
				return t.In(loc), nil
			}
			if tzid != "" {
				if tzLoc, err := time.LoadLocation(tzid); err == nil {
					return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), 0, tzLoc).In(loc), nil
				}
			}
			return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), 0, loc), nil
		}
	}

	return time.Time{}, fmt.Errorf("无法解析日期: %s", val)
}
