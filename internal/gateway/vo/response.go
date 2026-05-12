package vo

type HealthResp struct {
	Service string `json:"service"`
}

type PublicConfigResp struct {
	Env string `json:"env"`
}
