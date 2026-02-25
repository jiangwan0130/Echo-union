package repository

import (
	"context"

	"gorm.io/gorm"

	"echo-union/backend/internal/model"
)

// DepartmentRepository 部门数据访问接口
type DepartmentRepository interface {
	Create(ctx context.Context, dept *model.Department) error
	GetByID(ctx context.Context, id string) (*model.Department, error)
	GetByName(ctx context.Context, name string) (*model.Department, error)
	List(ctx context.Context) ([]model.Department, error)
	ListAll(ctx context.Context) ([]model.Department, error)
	Update(ctx context.Context, dept *model.Department) error
	Delete(ctx context.Context, id string, deletedBy string) error
	CountMembers(ctx context.Context, departmentID string) (int64, error)
}

// departmentRepo DepartmentRepository 的 GORM 实现
type departmentRepo struct {
	db *gorm.DB
}

// NewDepartmentRepo 创建 DepartmentRepository 实例
func NewDepartmentRepo(db *gorm.DB) DepartmentRepository {
	return &departmentRepo{db: db}
}

func (r *departmentRepo) Create(ctx context.Context, dept *model.Department) error {
	return r.db.WithContext(ctx).Create(dept).Error
}

func (r *departmentRepo) GetByID(ctx context.Context, id string) (*model.Department, error) {
	var dept model.Department
	err := r.db.WithContext(ctx).
		Where("department_id = ?", id).
		First(&dept).Error
	if err != nil {
		return nil, err
	}
	return &dept, nil
}

func (r *departmentRepo) List(ctx context.Context) ([]model.Department, error) {
	var depts []model.Department
	err := r.db.WithContext(ctx).
		Where("is_active = ?", true).
		Order("name ASC").
		Find(&depts).Error
	return depts, err
}

func (r *departmentRepo) Update(ctx context.Context, dept *model.Department) error {
	return r.db.WithContext(ctx).Save(dept).Error
}

func (r *departmentRepo) GetByName(ctx context.Context, name string) (*model.Department, error) {
	var dept model.Department
	err := r.db.WithContext(ctx).
		Where("name = ?", name).
		First(&dept).Error
	if err != nil {
		return nil, err
	}
	return &dept, nil
}

func (r *departmentRepo) ListAll(ctx context.Context) ([]model.Department, error) {
	var depts []model.Department
	err := r.db.WithContext(ctx).
		Order("name ASC").
		Find(&depts).Error
	return depts, err
}

func (r *departmentRepo) Delete(ctx context.Context, id string, deletedBy string) error {
	return r.db.WithContext(ctx).
		Model(&model.Department{}).
		Where("department_id = ?", id).
		Updates(map[string]interface{}{
			"deleted_by": deletedBy,
			"deleted_at": gorm.Expr("NOW()"),
		}).Error
}

func (r *departmentRepo) CountMembers(ctx context.Context, departmentID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.User{}).
		Where("department_id = ? AND deleted_at IS NULL", departmentID).
		Count(&count).Error
	return count, err
}

// [自证通过] internal/repository/department_repo.go
