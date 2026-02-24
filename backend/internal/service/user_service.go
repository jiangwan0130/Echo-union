package service

import (
	"context"
	"errors"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"echo-union/backend/internal/dto"
	"echo-union/backend/internal/repository"
)

// UserService 用户业务接口
type UserService interface {
	GetByID(ctx context.Context, id string) (*dto.UserResponse, error)
	List(ctx context.Context, page *dto.PaginationRequest) ([]dto.UserResponse, int64, error)
	// 📝 按需扩展: Create, Update, Delete, BatchImport, ChangePassword 等
}

type userService struct {
	repo   *repository.Repository
	logger *zap.Logger
}

// NewUserService 创建 UserService 实例
func NewUserService(repo *repository.Repository, logger *zap.Logger) UserService {
	return &userService{repo: repo, logger: logger}
}

func (s *userService) GetByID(ctx context.Context, id string) (*dto.UserResponse, error) {
	user, err := s.repo.User.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		s.logger.Error("查询用户失败", zap.String("id", id), zap.Error(err))
		return nil, err
	}

	var dept *dto.DepartmentResponse
	if user.Department != nil {
		dept = &dto.DepartmentResponse{
			ID:   user.Department.DepartmentID,
			Name: user.Department.Name,
		}
	}

	return &dto.UserResponse{
		ID:         user.UserID,
		Name:       user.Name,
		Email:      user.Email,
		StudentID:  user.StudentID,
		Role:       user.Role,
		Department: dept,
	}, nil
}

func (s *userService) List(ctx context.Context, page *dto.PaginationRequest) ([]dto.UserResponse, int64, error) {
	users, total, err := s.repo.User.List(ctx, page.GetOffset(), page.GetPageSize())
	if err != nil {
		s.logger.Error("列出用户失败", zap.Error(err))
		return nil, 0, err
	}

	result := make([]dto.UserResponse, 0, len(users))
	for _, u := range users {
		var dept *dto.DepartmentResponse
		if u.Department != nil {
			dept = &dto.DepartmentResponse{
				ID:   u.Department.DepartmentID,
				Name: u.Department.Name,
			}
		}
		result = append(result, dto.UserResponse{
			ID:         u.UserID,
			Name:       u.Name,
			Email:      u.Email,
			StudentID:  u.StudentID,
			Role:       u.Role,
			Department: dept,
		})
	}

	return result, total, nil
}

// [自证通过] internal/service/user_service.go
