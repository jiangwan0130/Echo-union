package service

import (
	"context"
	"errors"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"echo-union/backend/internal/dto"
	"echo-union/backend/internal/model"
	"echo-union/backend/internal/repository"
)

// ── 时间段模块业务错误 ──

var (
	ErrTimeSlotNotFound = errors.New("时间段不存在")
)

// TimeSlotService 时间段业务接口
type TimeSlotService interface {
	Create(ctx context.Context, req *dto.CreateTimeSlotRequest, callerID string) (*dto.TimeSlotResponse, error)
	GetByID(ctx context.Context, id string) (*dto.TimeSlotResponse, error)
	List(ctx context.Context, req *dto.TimeSlotListRequest) ([]dto.TimeSlotResponse, error)
	Update(ctx context.Context, id string, req *dto.UpdateTimeSlotRequest, callerID string) (*dto.TimeSlotResponse, error)
	Delete(ctx context.Context, id string, callerID string) error
}

type timeSlotService struct {
	repo   *repository.Repository
	logger *zap.Logger
}

// NewTimeSlotService 创建 TimeSlotService 实例
func NewTimeSlotService(repo *repository.Repository, logger *zap.Logger) TimeSlotService {
	return &timeSlotService{repo: repo, logger: logger}
}

// ────────────────────── Create ──────────────────────

func (s *timeSlotService) Create(ctx context.Context, req *dto.CreateTimeSlotRequest, callerID string) (*dto.TimeSlotResponse, error) {
	// 如果指定了学期ID，验证学期存在
	if req.SemesterID != nil {
		if _, err := s.repo.Semester.GetByID(ctx, *req.SemesterID); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, ErrSemesterNotFound
			}
			return nil, err
		}
	}

	slot := &model.TimeSlot{
		Name:       req.Name,
		SemesterID: req.SemesterID,
		StartTime:  req.StartTime,
		EndTime:    req.EndTime,
		DayOfWeek:  req.DayOfWeek,
		IsActive:   true,
	}
	slot.CreatedBy = &callerID
	slot.UpdatedBy = &callerID

	if err := s.repo.TimeSlot.Create(ctx, slot); err != nil {
		s.logger.Error("创建时间段失败", zap.Error(err))
		return nil, err
	}

	// 重新加载以获取关联
	created, err := s.repo.TimeSlot.GetByID(ctx, slot.TimeSlotID)
	if err != nil {
		return nil, err
	}

	return s.toTimeSlotResponse(created), nil
}

// ────────────────────── GetByID ──────────────────────

func (s *timeSlotService) GetByID(ctx context.Context, id string) (*dto.TimeSlotResponse, error) {
	slot, err := s.repo.TimeSlot.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTimeSlotNotFound
		}
		s.logger.Error("查询时间段失败", zap.String("id", id), zap.Error(err))
		return nil, err
	}

	return s.toTimeSlotResponse(slot), nil
}

// ────────────────────── List ──────────────────────

func (s *timeSlotService) List(ctx context.Context, req *dto.TimeSlotListRequest) ([]dto.TimeSlotResponse, error) {
	slots, err := s.repo.TimeSlot.List(ctx, req.SemesterID, req.DayOfWeek)
	if err != nil {
		s.logger.Error("列出时间段失败", zap.Error(err))
		return nil, err
	}

	result := make([]dto.TimeSlotResponse, 0, len(slots))
	for i := range slots {
		result = append(result, *s.toTimeSlotResponse(&slots[i]))
	}

	return result, nil
}

// ────────────────────── Update ──────────────────────

func (s *timeSlotService) Update(ctx context.Context, id string, req *dto.UpdateTimeSlotRequest, callerID string) (*dto.TimeSlotResponse, error) {
	slot, err := s.repo.TimeSlot.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTimeSlotNotFound
		}
		s.logger.Error("查询时间段失败", zap.String("id", id), zap.Error(err))
		return nil, err
	}

	if req.Name != nil {
		slot.Name = *req.Name
	}
	if req.StartTime != nil {
		slot.StartTime = *req.StartTime
	}
	if req.EndTime != nil {
		slot.EndTime = *req.EndTime
	}
	if req.DayOfWeek != nil {
		slot.DayOfWeek = *req.DayOfWeek
	}
	if req.IsActive != nil {
		slot.IsActive = *req.IsActive
	}

	slot.UpdatedBy = &callerID

	if err := s.repo.TimeSlot.Update(ctx, slot); err != nil {
		s.logger.Error("更新时间段失败", zap.String("id", id), zap.Error(err))
		return nil, err
	}

	return s.toTimeSlotResponse(slot), nil
}

// ────────────────────── Delete ──────────────────────

func (s *timeSlotService) Delete(ctx context.Context, id string, callerID string) error {
	_, err := s.repo.TimeSlot.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrTimeSlotNotFound
		}
		s.logger.Error("查询时间段失败", zap.String("id", id), zap.Error(err))
		return err
	}

	if err := s.repo.TimeSlot.Delete(ctx, id, callerID); err != nil {
		s.logger.Error("删除时间段失败", zap.String("id", id), zap.Error(err))
		return err
	}

	return nil
}

// ── 内部辅助方法 ──

func (s *timeSlotService) toTimeSlotResponse(slot *model.TimeSlot) *dto.TimeSlotResponse {
	resp := &dto.TimeSlotResponse{
		ID:         slot.TimeSlotID,
		Name:       slot.Name,
		SemesterID: slot.SemesterID,
		StartTime:  slot.StartTime,
		EndTime:    slot.EndTime,
		DayOfWeek:  slot.DayOfWeek,
		IsActive:   slot.IsActive,
		CreatedAt:  slot.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:  slot.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}

	if slot.Semester != nil {
		resp.Semester = &dto.SemesterBrief{
			ID:   slot.Semester.SemesterID,
			Name: slot.Semester.Name,
		}
	}

	return resp
}
