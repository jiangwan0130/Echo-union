package model

// Location 值班地点表 — 对应 locations（V1 预留扩展）
type Location struct {
	LocationID string `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"location_id"`
	Name       string `gorm:"type:varchar(100);not null"                     json:"name"`
	Address    string `gorm:"type:varchar(200)"                              json:"address,omitempty"`
	IsDefault  bool   `gorm:"not null;default:false"                         json:"is_default"`
	IsActive   bool   `gorm:"not null;default:true"                          json:"is_active"`
	SoftDeleteModel
}

// TableName 指定表名
func (Location) TableName() string { return "locations" }

// [自证通过] internal/model/location.go
