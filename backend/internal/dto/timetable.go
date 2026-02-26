package dto

import (
	"fmt"
	"time"
)

// ── ICS 导入 ──

// ImportICSRequest ICS 导入请求（用于 URL 方式）
type ImportICSRequest struct {
	URL        string `json:"url" binding:"omitempty,url"`
	SemesterID string `json:"semester_id" binding:"omitempty,uuid"`
}

// ImportICSResponse ICS 导入响应
type ImportICSResponse struct {
	ImportedCount int                   `json:"imported_count"`
	Events        []ImportedCourseEvent `json:"events"`
}

// ImportedCourseEvent 导入的课程事件
type ImportedCourseEvent struct {
	Name      string `json:"name"`
	DayOfWeek int    `json:"day_of_week"`
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
	Weeks     []int  `json:"weeks"`
}

// ── 不可用时间 ──

// CreateUnavailableTimeRequest 添加不可用时间请求
type CreateUnavailableTimeRequest struct {
	DayOfWeek    int     `json:"day_of_week" binding:"required,min=1,max=7"`
	StartTime    string  `json:"start_time" binding:"required"`
	EndTime      string  `json:"end_time" binding:"required"`
	Reason       string  `json:"reason" binding:"omitempty,max=200"`
	RepeatType   string  `json:"repeat_type" binding:"omitempty,oneof=weekly biweekly once"`
	SpecificDate *string `json:"specific_date" binding:"omitempty"` // YYYY-MM-DD，仅 once 类型必填
	WeekType     string  `json:"week_type" binding:"omitempty,oneof=all odd even"`
	SemesterID   string  `json:"semester_id" binding:"omitempty,uuid"`
}

// Validate 校验业务规则（repeat_type 与 week_type/specific_date 的联动约束）
func (r *CreateUnavailableTimeRequest) Validate() error {
	rt := r.RepeatType
	if rt == "" {
		rt = "weekly" // 默认值
	}
	switch rt {
	case "once":
		if r.SpecificDate == nil || *r.SpecificDate == "" {
			return fmt.Errorf("单次类型必须指定 specific_date")
		}
		if r.WeekType != "" && r.WeekType != "all" {
			return fmt.Errorf("单次类型的 week_type 必须为 all")
		}
	case "weekly":
		if r.SpecificDate != nil && *r.SpecificDate != "" {
			return fmt.Errorf("每周重复类型不应指定 specific_date")
		}
	case "biweekly":
		if r.SpecificDate != nil && *r.SpecificDate != "" {
			return fmt.Errorf("双周重复类型不应指定 specific_date")
		}
		if r.WeekType != "odd" && r.WeekType != "even" {
			return fmt.Errorf("双周重复类型的 week_type 必须为 odd 或 even")
		}
	}
	return nil
}

// UpdateUnavailableTimeRequest 更新不可用时间请求
type UpdateUnavailableTimeRequest struct {
	DayOfWeek    *int    `json:"day_of_week" binding:"omitempty,min=1,max=7"`
	StartTime    *string `json:"start_time" binding:"omitempty"`
	EndTime      *string `json:"end_time" binding:"omitempty"`
	Reason       *string `json:"reason" binding:"omitempty,max=200"`
	RepeatType   *string `json:"repeat_type" binding:"omitempty,oneof=weekly biweekly once"`
	SpecificDate *string `json:"specific_date" binding:"omitempty"`
	WeekType     *string `json:"week_type" binding:"omitempty,oneof=all odd even"`
}

// UnavailableTimeResponse 不可用时间响应
type UnavailableTimeResponse struct {
	ID           string     `json:"id"`
	DayOfWeek    int        `json:"day_of_week"`
	StartTime    string     `json:"start_time"`
	EndTime      string     `json:"end_time"`
	Reason       string     `json:"reason"`
	RepeatType   string     `json:"repeat_type"`
	SpecificDate *time.Time `json:"specific_date,omitempty"`
	WeekType     string     `json:"week_type"`
}

// ── 我的时间表 ──

// MyTimetableRequest 查询时间表请求
type MyTimetableRequest struct {
	SemesterID string `form:"semester_id" binding:"omitempty,uuid"`
}

// MyTimetableResponse 我的时间表响应
type MyTimetableResponse struct {
	Courses      []CourseResponse          `json:"courses"`
	Unavailable  []UnavailableTimeResponse `json:"unavailable"`
	SubmitStatus string                    `json:"submit_status"`
	SubmittedAt  *time.Time                `json:"submitted_at,omitempty"`
}

// CourseResponse 课表条目响应
type CourseResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	DayOfWeek int    `json:"day_of_week"`
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
	WeekType  string `json:"week_type"`
	Weeks     []int  `json:"weeks"`
	Source    string `json:"source"`
}

// ── 提交时间表 ──

// SubmitTimetableRequest 提交时间表请求
type SubmitTimetableRequest struct {
	SemesterID string `json:"semester_id" binding:"omitempty,uuid"`
}

// SubmitTimetableResponse 提交时间表响应
type SubmitTimetableResponse struct {
	SubmitStatus string     `json:"submit_status"`
	SubmittedAt  *time.Time `json:"submitted_at"`
}

// ── 提交进度 ──

// TimetableProgressResponse 全局提交进度响应
type TimetableProgressResponse struct {
	Total       int64                    `json:"total"`
	Submitted   int64                    `json:"submitted"`
	Progress    float64                  `json:"progress"` // 0-100
	Departments []DepartmentProgressItem `json:"departments"`
}

// DepartmentProgressItem 部门提交进度条目
type DepartmentProgressItem struct {
	DepartmentID   string  `json:"department_id"`
	DepartmentName string  `json:"department_name"`
	Total          int     `json:"total"`
	Submitted      int     `json:"submitted"`
	Progress       float64 `json:"progress"`
}

// DepartmentProgressResponse 单部门提交进度响应
type DepartmentProgressResponse struct {
	DepartmentID   string                   `json:"department_id"`
	DepartmentName string                   `json:"department_name"`
	Total          int                      `json:"total"`
	Submitted      int                      `json:"submitted"`
	Progress       float64                  `json:"progress"`
	Members        []DepartmentMemberStatus `json:"members"`
}

// DepartmentMemberStatus 部门成员提交状态
type DepartmentMemberStatus struct {
	UserID          string     `json:"user_id"`
	Name            string     `json:"name"`
	StudentID       string     `json:"student_id"`
	TimetableStatus string     `json:"timetable_status"`
	SubmittedAt     *time.Time `json:"submitted_at,omitempty"`
}
