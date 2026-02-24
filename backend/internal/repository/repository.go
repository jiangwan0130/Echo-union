package repository

import "gorm.io/gorm"

// Repository 所有 Repository 的聚合入口
type Repository struct {
	User       UserRepository
	Department DepartmentRepository
	// 📝 后续按模块扩展其他 Repository 接口
}

// NewRepository 创建 Repository 聚合
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{
		User:       NewUserRepo(db),
		Department: NewDepartmentRepo(db),
	}
}

// [自证通过] internal/repository/repository.go
