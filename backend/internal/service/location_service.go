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

// ── 地点模块业务错误 ──

var (
	ErrLocationNotFound = errors.New("地点不存在")
)

// LocationService 地点业务接口
type LocationService interface {
	Create(ctx context.Context, req *dto.CreateLocationRequest, callerID string) (*dto.LocationResponse, error)
	GetByID(ctx context.Context, id string) (*dto.LocationResponse, error)
	List(ctx context.Context, req *dto.LocationListRequest) ([]dto.LocationResponse, error)
	Update(ctx context.Context, id string, req *dto.UpdateLocationRequest, callerID string) (*dto.LocationResponse, error)
	Delete(ctx context.Context, id string, callerID string) error
}

type locationService struct {
	repo   *repository.Repository
	logger *zap.Logger
}

// NewLocationService 创建 LocationService 实例
func NewLocationService(repo *repository.Repository, logger *zap.Logger) LocationService {
	return &locationService{repo: repo, logger: logger}
}

// ────────────────────── Create ──────────────────────

func (s *locationService) Create(ctx context.Context, req *dto.CreateLocationRequest, callerID string) (*dto.LocationResponse, error) {
	loc := &model.Location{
		Name:      req.Name,
		Address:   req.Address,
		IsDefault: req.IsDefault,
		IsActive:  true,
	}
	loc.CreatedBy = &callerID
	loc.UpdatedBy = &callerID

	if err := s.repo.Location.Create(ctx, loc); err != nil {
		s.logger.Error("创建地点失败", zap.Error(err))
		return nil, err
	}

	return s.toLocationResponse(loc), nil
}

// ────────────────────── GetByID ──────────────────────

func (s *locationService) GetByID(ctx context.Context, id string) (*dto.LocationResponse, error) {
	loc, err := s.repo.Location.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrLocationNotFound
		}
		s.logger.Error("查询地点失败", zap.String("id", id), zap.Error(err))
		return nil, err
	}

	return s.toLocationResponse(loc), nil
}

// ────────────────────── List ──────────────────────

func (s *locationService) List(ctx context.Context, req *dto.LocationListRequest) ([]dto.LocationResponse, error) {
	locations, err := s.repo.Location.List(ctx, req.IncludeInactive)
	if err != nil {
		s.logger.Error("列出地点失败", zap.Error(err))
		return nil, err
	}

	result := make([]dto.LocationResponse, 0, len(locations))
	for i := range locations {
		result = append(result, *s.toLocationResponse(&locations[i]))
	}

	return result, nil
}

// ────────────────────── Update ──────────────────────

func (s *locationService) Update(ctx context.Context, id string, req *dto.UpdateLocationRequest, callerID string) (*dto.LocationResponse, error) {
	loc, err := s.repo.Location.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrLocationNotFound
		}
		s.logger.Error("查询地点失败", zap.String("id", id), zap.Error(err))
		return nil, err
	}

	if req.Name != nil {
		loc.Name = *req.Name
	}
	if req.Address != nil {
		loc.Address = *req.Address
	}
	if req.IsDefault != nil {
		loc.IsDefault = *req.IsDefault
	}
	if req.IsActive != nil {
		loc.IsActive = *req.IsActive
	}

	loc.UpdatedBy = &callerID

	if err := s.repo.Location.Update(ctx, loc); err != nil {
		s.logger.Error("更新地点失败", zap.String("id", id), zap.Error(err))
		return nil, err
	}

	return s.toLocationResponse(loc), nil
}

// ────────────────────── Delete ──────────────────────

func (s *locationService) Delete(ctx context.Context, id string, callerID string) error {
	_, err := s.repo.Location.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrLocationNotFound
		}
		s.logger.Error("查询地点失败", zap.String("id", id), zap.Error(err))
		return err
	}

	if err := s.repo.Location.Delete(ctx, id, callerID); err != nil {
		s.logger.Error("删除地点失败", zap.String("id", id), zap.Error(err))
		return err
	}

	return nil
}

// ── 内部辅助方法 ──

func (s *locationService) toLocationResponse(loc *model.Location) *dto.LocationResponse {
	return &dto.LocationResponse{
		ID:        loc.LocationID,
		Name:      loc.Name,
		Address:   loc.Address,
		IsDefault: loc.IsDefault,
		IsActive:  loc.IsActive,
		CreatedAt: loc.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt: loc.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
}
