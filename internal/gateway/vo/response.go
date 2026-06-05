package vo

// HealthResp 网关健康检查响应体。
type HealthResp struct {
	Service string `json:"service"`
}

// PublicConfigResp 网关公开配置响应（如运行环境标识）。
type PublicConfigResp struct {
	Env string `json:"env"`
}
