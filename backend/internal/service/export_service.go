package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/xuri/excelize/v2"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"echo-union/backend/internal/repository"
)

// ── 导出模块业务错误 ──

var (
	ErrExportNoSchedule   = errors.New("该学期暂无排班表")
	ErrExportNoItems      = errors.New("排班表中无排班项")
	ErrExportGenerateFail = errors.New("生成 Excel 文件失败")
)

// ExportService 导出业务接口
//
// 设计说明：
//   - 一期仅实现排班表导出为 Excel (.xlsx)
//   - 签到统计导出依赖签到模块，归入二期
//   - 导出以 bytes.Buffer 返回，由 Handler 层设置 HTTP 响应头后写入 Response
//   - Excel 格式：按周次分 Sheet，每个 Sheet 按 day_of_week 列 × time_slot 行呈现
type ExportService interface {
	// ExportSchedule 导出排班表为 Excel
	ExportSchedule(ctx context.Context, semesterID string) (*bytes.Buffer, string, error)
}

type exportService struct {
	repo   *repository.Repository
	logger *zap.Logger
}

// NewExportService 创建 ExportService 实例
func NewExportService(repo *repository.Repository, logger *zap.Logger) ExportService {
	return &exportService{repo: repo, logger: logger}
}

// ═══════════════════════════════════════════════════════════
// ExportSchedule — 导出排班表为 Excel
// ═══════════════════════════════════════════════════════════
//
// 输出格式：
//   - Sheet "第1周" / "第2周"（按 week_number 分）
//   - 行头：时间段名称（按 day_of_week + start_time 排序）
//   - 列头：周一 ~ 周五
//   - 单元格：成员姓名 (部门名)
//
// 返回值：buf（Excel 内容）, filename（建议文件名）, error

func (s *exportService) ExportSchedule(ctx context.Context, semesterID string) (*bytes.Buffer, string, error) {
	// 1. 查询排班表
	schedule, err := s.repo.Schedule.GetBySemester(ctx, semesterID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, "", ErrExportNoSchedule
		}
		s.logger.Error("查询排班表失败", zap.Error(err))
		return nil, "", err
	}

	// 2. 查询排班项
	items, err := s.repo.ScheduleItem.ListBySchedule(ctx, schedule.ScheduleID)
	if err != nil {
		s.logger.Error("查询排班明细失败", zap.Error(err))
		return nil, "", err
	}
	if len(items) == 0 {
		return nil, "", ErrExportNoItems
	}

	// 3. 获取学期名称
	semesterName := semesterID
	if schedule.Semester != nil {
		semesterName = schedule.Semester.Name
	}

	// 4. 构建数据索引: "weekNumber:dayOfWeek:slotName:startTime" → cellText
	//    以及收集唯一时间段（按天分组）
	type slotKey struct {
		dayOfWeek int
		name      string
		startTime string
		endTime   string
	}

	itemIndex := make(map[string]string) // "wn:dow:name:start" → cellText
	slotsByDay := make(map[int][]slotKey)
	slotSeen := make(map[string]bool)

	for _, item := range items {
		if item.TimeSlot == nil {
			continue
		}
		ts := item.TimeSlot

		// 构建单元格文本
		cellText := "未分配"
		if item.Member != nil {
			cellText = item.Member.Name
			if item.Member.Department != nil {
				cellText += " (" + item.Member.Department.Name + ")"
			}
		}

		key := fmt.Sprintf("%d:%d:%s:%s", item.WeekNumber, ts.DayOfWeek, ts.Name, ts.StartTime)
		itemIndex[key] = cellText

		// 记录唯一时间段
		slotID := fmt.Sprintf("%d:%s", ts.DayOfWeek, ts.TimeSlotID)
		if !slotSeen[slotID] {
			slotSeen[slotID] = true
			slotsByDay[ts.DayOfWeek] = append(slotsByDay[ts.DayOfWeek], slotKey{
				dayOfWeek: ts.DayOfWeek,
				name:      ts.Name,
				startTime: ts.StartTime,
				endTime:   ts.EndTime,
			})
		}
	}

	// 5. 收集并排序所有时间段（按 dayOfWeek + startTime）
	var allSlots []slotKey
	for _, slots := range slotsByDay {
		allSlots = append(allSlots, slots...)
	}
	sort.Slice(allSlots, func(i, j int) bool {
		if allSlots[i].dayOfWeek != allSlots[j].dayOfWeek {
			return allSlots[i].dayOfWeek < allSlots[j].dayOfWeek
		}
		return allSlots[i].startTime < allSlots[j].startTime
	})

	// 按天分组的有序时间段（去重后）：用于构建行
	type daySlots struct {
		dayOfWeek int
		slots     []slotKey
	}
	dayOrder := []int{1, 2, 3, 4, 5}
	daySlotsMap := make(map[int][]slotKey)
	for _, sl := range allSlots {
		daySlotsMap[sl.dayOfWeek] = append(daySlotsMap[sl.dayOfWeek], sl)
	}

	// 6. 获取所有唯一时间段用于构建行
	// 行结构: 每天有多个时间段行
	type rowDef struct {
		dayOfWeek int
		slotName  string
		startTime string
		endTime   string
	}
	var rows []rowDef
	for _, dow := range dayOrder {
		for _, sl := range daySlotsMap[dow] {
			rows = append(rows, rowDef{
				dayOfWeek: dow,
				slotName:  sl.name,
				startTime: sl.startTime,
				endTime:   sl.endTime,
			})
		}
	}

	// 重新构建：以 (星期, 时间段) 为行，week_number 为列
	// 表头: | 星期 | 时间段 | 时间 | 第1周 | 第2周 |
	dayNames := map[int]string{1: "周一", 2: "周二", 3: "周三", 4: "周四", 5: "周五"}

	// 找出所有 weekNumbers
	weekSet := make(map[int]bool)
	for _, item := range items {
		weekSet[item.WeekNumber] = true
	}
	var weekNumbers []int
	for wn := range weekSet {
		weekNumbers = append(weekNumbers, wn)
	}
	sort.Ints(weekNumbers)
	if len(weekNumbers) == 0 {
		weekNumbers = []int{1, 2}
	}

	// 7. 生成 Excel
	f := excelize.NewFile()
	defer f.Close()

	sheetName := "排班表"
	idx, _ := f.NewSheet(sheetName)
	f.SetActiveSheet(idx)
	// 删除默认 Sheet1
	f.DeleteSheet("Sheet1")

	// 设置列宽
	f.SetColWidth(sheetName, "A", "A", 8)
	f.SetColWidth(sheetName, "B", "B", 14)
	f.SetColWidth(sheetName, "C", "C", 18)
	for i := range weekNumbers {
		col, _ := excelize.ColumnNumberToName(4 + i)
		f.SetColWidth(sheetName, col, col, 22)
	}

	// 样式
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 11},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#4472C4"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})

	// 标题行
	f.SetCellValue(sheetName, "A1", fmt.Sprintf("%s — 排班表", semesterName))
	f.MergeCell(sheetName, "A1", fmt.Sprintf("%s1", colName(3+len(weekNumbers))))
	titleCell, _ := excelize.CoordinatesToCellName(1, 1)
	f.SetCellStyle(sheetName, titleCell, titleCell, headerStyle)

	// 表头
	row := 2
	f.SetCellValue(sheetName, cell("A", row), "星期")
	f.SetCellValue(sheetName, cell("B", row), "时间段")
	f.SetCellValue(sheetName, cell("C", row), "时间")
	for i, wn := range weekNumbers {
		f.SetCellValue(sheetName, cell(colName(3+i), row), fmt.Sprintf("第%d周", wn))
	}

	// 数据行
	row = 3
	for _, rd := range rows {
		f.SetCellValue(sheetName, cell("A", row), dayNames[rd.dayOfWeek])
		f.SetCellValue(sheetName, cell("B", row), rd.slotName)
		f.SetCellValue(sheetName, cell("C", row), fmt.Sprintf("%s-%s", rd.startTime, rd.endTime))

		for i, wn := range weekNumbers {
			key := fmt.Sprintf("%d:%d:%s:%s", wn, rd.dayOfWeek, rd.slotName, rd.startTime)
			if text, ok := itemIndex[key]; ok {
				f.SetCellValue(sheetName, cell(colName(3+i), row), text)
			} else {
				f.SetCellValue(sheetName, cell(colName(3+i), row), "-")
			}
		}
		row++
	}

	// 写入 buffer
	buf := new(bytes.Buffer)
	if err := f.Write(buf); err != nil {
		s.logger.Error("写入 Excel 失败", zap.Error(err))
		return nil, "", ErrExportGenerateFail
	}

	filename := fmt.Sprintf("排班表_%s.xlsx", semesterName)
	return buf, filename, nil
}

// ── 辅助函数 ──

func colName(idx int) string {
	name, _ := excelize.ColumnNumberToName(idx + 1)
	return name
}

func cell(col string, row int) string {
	return fmt.Sprintf("%s%d", col, row)
}
