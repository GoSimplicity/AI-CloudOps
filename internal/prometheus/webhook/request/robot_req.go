package request

// RobotTenantAccessTokenReq 表示获取租户访问令牌的请求结构体
type RobotTenantAccessTokenReq struct {
	AppID     string `json:"app_id"`     // 应用 ID
	AppSecret string `json:"app_secret"` // 应用密钥
}

// RobotTenantAccessTokenRes 表示租户访问令牌请求的响应结构体
type RobotTenantAccessTokenRes struct {
	Code              int    `json:"code"`                // 响应代码
	Expire            int    `json:"expire"`              // 令牌过期时间（秒）
	Message           string `json:"message"`             // 响应消息
	TenantAccessToken string `json:"tenant_access_token"` // 获取到的租户访问令牌
}
