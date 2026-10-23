package model

// UserPermissionsVersion 是 GET /api/user/permissions 响应格式的版本号。
const UserPermissionsVersion = 2

// UserPermissionsResp 是 GET /api/user/permissions 的响应数据（契约见 docs/api/user-permissions-v2.md）。
// 仅用于接口响应，不是数据库模型，不参与 AutoMigrate。
type UserPermissionsResp struct {
	Version     int      `json:"version"`
	SuperAdmin  bool     `json:"super_admin"`
	Permissions []string `json:"permissions"`
}
