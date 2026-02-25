package dto

// ── 地点模块 DTO ──

// CreateLocationRequest 创建地点请求
type CreateLocationRequest struct {
	Name      string `json:"name"       binding:"required,min=2,max=100"`
	Address   string `json:"address"    binding:"omitempty,max=200"`
	IsDefault bool   `json:"is_default"`
}

// UpdateLocationRequest 更新地点请求
type UpdateLocationRequest struct {
	Name      *string `json:"name"       binding:"omitempty,min=2,max=100"`
	Address   *string `json:"address"    binding:"omitempty,max=200"`
	IsDefault *bool   `json:"is_default"`
	IsActive  *bool   `json:"is_active"`
}

// LocationListRequest 地点列表查询参数
type LocationListRequest struct {
	IncludeInactive bool `form:"include_inactive"`
}

// LocationResponse 地点信息响应
type LocationResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Address   string `json:"address,omitempty"`
	IsDefault bool   `json:"is_default"`
	IsActive  bool   `json:"is_active"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}
