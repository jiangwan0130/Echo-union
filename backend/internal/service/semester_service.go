package service

import (
	"context"
	"errors"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"echo-union/backend/internal/dto"
	"echo-union/backend/internal/model"
	"echo-union/backend/internal/repository"
)

// ── 学期模块业务错误 ──

var (
	ErrSemesterNotFound    = errors.New("学期不存在")
	ErrSemesterDateInvalid = errors.New("学期结束日期必须晚于开始日期")
	ErrSemesterDateOverlap = errors.New("学期日期与已有学期重叠")
)

// SemesterService 学期业务接口
type SemesterService interface {
	Create(ctx context.Context, req *dto.CreateSemesterRequest, callerID string) (*dto.SemesterResponse, error)
	GetByID(ctx context.Context, id string) (*dto.SemesterResponse, error)
	GetCurrent(ctx context.Context) (*dto.SemesterResponse, error)
	List(ctx context.Context) ([]dto.SemesterResponse, error)
	Update(ctx context.Context, id string, req *dto.UpdateSemesterRequest, callerID string) (*dto.SemesterResponse, error)
	Activate(ctx context.Context, id string, callerID string) error
	Delete(ctx context.Context, id string, callerID string) error
}

type semesterService struct {
	repo   *repository.Repository
	logger *zap.Logger
}

// NewSemesterService 创建 SemesterService 实例
func NewSemesterService(repo *repository.Repository, logger *zap.Logger) SemesterService {
	return &semesterService{repo: repo, logger: logger}
}

// ────────────────────── Create ──────────────────────

func (s *semesterService) Create(ctx context.Context, req *dto.CreateSemesterRequest, callerID string) (*dto.SemesterResponse, error) {
	startDate, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		return nil, ErrSemesterDateInvalid
	}
	endDate, err := time.Parse("2006-01-02", req.EndDate)
	if err != nil {
		return nil, ErrSemesterDateInvalid
	}
	if !endDate.After(startDate) {
		return nil, ErrSemesterDateInvalid
	}

	semester := &model.Semester{
		Name:          req.Name,
		StartDate:     startDate,
		EndDate:       endDate,
		FirstWeekType: req.FirstWeekType,
		IsActive:      false,
		Status:        "active",
	}
	semester.CreatedBy = &callerID
	semester.UpdatedBy = &callerID

	if err := s.repo.Semester.Create(ctx, semester); err != nil {
		s.logger.Error("创建学期失败", zap.Error(err))
		return nil, err
	}

	return s.toSemesterResponse(semester), nil
}

// ────────────────────── GetByID ──────────────────────

func (s *semesterService) GetByID(ctx context.Context, id string) (*dto.SemesterResponse, error) {
	semester, err := s.repo.Semester.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSemesterNotFound
		}
		s.logger.Error("查询学期失败", zap.String("id", id), zap.Error(err))
		return nil, err
	}

	return s.toSemesterResponse(semester), nil
}

// ────────────────────── GetCurrent ──────────────────────

func (s *semesterService) GetCurrent(ctx context.Context) (*dto.SemesterResponse, error) {
	semester, err := s.repo.Semester.GetCurrent(ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSemesterNotFound
		}
		s.logger.Error("查询当前学期失败", zap.Error(err))
		return nil, err
	}

	return s.toSemesterResponse(semester), nil
}

// ────────────────────── List ──────────────────────

func (s *semesterService) List(ctx context.Context) ([]dto.SemesterResponse, error) {
	semesters, err := s.repo.Semester.List(ctx)
	if err != nil {
		s.logger.Error("列出学期失败", zap.Error(err))
		return nil, err
	}

	result := make([]dto.SemesterResponse, 0, len(semesters))
	for i := range semesters {
		result = append(result, *s.toSemesterResponse(&semesters[i]))
	}

	return result, nil
}

// ────────────────────── Update ──────────────────────

func (s *semesterService) Update(ctx context.Context, id string, req *dto.UpdateSemesterRequest, callerID string) (*dto.SemesterResponse, error) {
	semester, err := s.repo.Semester.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSemesterNotFound
		}
		s.logger.Error("查询学期失败", zap.String("id", id), zap.Error(err))
		return nil, err
	}

	if req.Name != nil {
		semester.Name = *req.Name
	}
	if req.StartDate != nil {
		startDate, err := time.Parse("2006-01-02", *req.StartDate)
		if err != nil {
			return nil, ErrSemesterDateInvalid
		}
		semester.StartDate = startDate
	}
	if req.EndDate != nil {
		endDate, err := time.Parse("2006-01-02", *req.EndDate)
		if err != nil {
			return nil, ErrSemesterDateInvalid
		}
		semester.EndDate = endDate
	}
	if !semester.EndDate.After(semester.StartDate) {
		return nil, ErrSemesterDateInvalid
	}
	if req.FirstWeekType != nil {
		semester.FirstWeekType = *req.FirstWeekType
	}
	if req.Status != nil {
		semester.Status = *req.Status
	}

	semester.UpdatedBy = &callerID

	if err := s.repo.Semester.Update(ctx, semester); err != nil {
		s.logger.Error("更新学期失败", zap.String("id", id), zap.Error(err))
		return nil, err
	}

	return s.toSemesterResponse(semester), nil
}

// ────────────────────── Activate ──────────────────────

func (s *semesterService) Activate(ctx context.Context, id string, callerID string) error {
	semester, err := s.repo.Semester.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrSemesterNotFound
		}
		s.logger.Error("查询学期失败", zap.String("id", id), zap.Error(err))
		return err
	}

	// 使用事务保证 ClearActive + Update 的原子性
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		s.logger.Error("开启事务失败", zap.Error(err))
		return err
	}
	defer func() {
		if r := recover(); r != nil {
			if tx != nil {
				tx.Rollback()
			}
			panic(r)
		}
	}()

	txRepo := s.repo.WithTx(tx)

	// 先将所有学期置为非活动
	if err := txRepo.Semester.ClearActive(ctx); err != nil {
		if tx != nil {
			tx.Rollback()
		}
		s.logger.Error("清除活动学期失败", zap.Error(err))
		return err
	}

	// 设置目标学期为活动
	semester.IsActive = true
	semester.UpdatedBy = &callerID

	if err := txRepo.Semester.Update(ctx, semester); err != nil {
		if tx != nil {
			tx.Rollback()
		}
		s.logger.Error("激活学期失败", zap.String("id", id), zap.Error(err))
		return err
	}

	if tx != nil {
		if err := tx.Commit().Error; err != nil {
			s.logger.Error("提交事务失败", zap.Error(err))
			return err
		}
	}

	return nil
}

// ────────────────────── Delete ──────────────────────

func (s *semesterService) Delete(ctx context.Context, id string, callerID string) error {
	_, err := s.repo.Semester.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrSemesterNotFound
		}
		s.logger.Error("查询学期失败", zap.String("id", id), zap.Error(err))
		return err
	}

	if err := s.repo.Semester.Delete(ctx, id, callerID); err != nil {
		s.logger.Error("删除学期失败", zap.String("id", id), zap.Error(err))
		return err
	}

	return nil
}

// ── 内部辅助方法 ──

func (s *semesterService) toSemesterResponse(semester *model.Semester) *dto.SemesterResponse {
	return &dto.SemesterResponse{
		ID:            semester.SemesterID,
		Name:          semester.Name,
		StartDate:     semester.StartDate.Format("2006-01-02"),
		EndDate:       semester.EndDate.Format("2006-01-02"),
		FirstWeekType: semester.FirstWeekType,
		IsActive:      semester.IsActive,
		Status:        semester.Status,
		CreatedAt:     semester.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:     semester.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
}
