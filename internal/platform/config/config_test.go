package config

import (
	"os"
	"testing"
)

func TestDefaultConfig_ports(t *testing.T) {
	cfg := defaultConfig()
	if cfg.GatewayServicePort != "8180" {
		t.Fatalf("gateway port=%s", cfg.GatewayServicePort)
	}
	if cfg.MemberServicePort != "8181" {
		t.Fatalf("member port=%s", cfg.MemberServicePort)
	}
}

func TestMerge_gatewayServicePort(t *testing.T) {
	dst := defaultConfig()
	src := &AppConfig{GatewayServicePort: "9191"}
	merge(dst, src)
	if dst.GatewayServicePort != "9191" {
		t.Fatalf("merge failed: %s", dst.GatewayServicePort)
	}
}

func TestOverrideWithEnv_gatewayServicePort(t *testing.T) {
	t.Setenv("GATEWAY_SERVICE_PORT", "8280")
	cfg := defaultConfig()
	overrideWithEnv(cfg)
	if cfg.GatewayServicePort != "8280" {
		t.Fatalf("env override failed: %s", cfg.GatewayServicePort)
	}
	os.Unsetenv("GATEWAY_SERVICE_PORT")
}
