package config

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type AppConfig struct {
	APIGatewayPort        string `json:"api_gateway_port"`
	MemberServicePort     string `json:"member_service_port"`
	MemberServiceGRPCPort string `json:"member_service_grpc_port"`

	MemberServiceURL      string `json:"member_service_url"`
	MemberServiceGRPCAddr string `json:"member_service_grpc_addr"`
	GatewayUseMemberGRPC  bool   `json:"gateway_use_member_grpc"`

	MySQLDSN      string `json:"mysql_dsn"`
	RedisAddr     string `json:"redis_addr"`
	RedisPassword string `json:"redis_password"`
	RedisDB       int    `json:"redis_db"`

	CORSAllowOrigin string `json:"cors_allow_origin"`

	ConsulEnabled    bool   `json:"consul_enabled"`
	ConsulAddress    string `json:"consul_address"`
	ConsulDatacenter string `json:"consul_datacenter"`
	ServiceHost      string `json:"service_host"`
}

func Load() (*AppConfig, error) {
	cfg := defaultConfig()

	if err := loadDotEnv("configs/.env"); err != nil {
		return nil, err
	}
	if err := loadDotEnv(".env"); err != nil {
		return nil, err
	}

	filePath := resolveConfigPath()
	if fileCfg, err := loadFile(filePath); err == nil {
		merge(cfg, fileCfg)
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}

	overrideWithEnv(cfg)
	return cfg, nil
}

func resolveConfigPath() string {
	if path := os.Getenv("CONFIG_FILE"); path != "" {
		return path
	}
	env := getenv("APP_ENV", "dev")
	return fmt.Sprintf("configs/app.%s.json", env)
}

func defaultConfig() *AppConfig {
	return &AppConfig{
		APIGatewayPort:        "8180",
		MemberServicePort:     "8181",
		MemberServiceGRPCPort: "9181",

		MemberServiceURL:      "http://127.0.0.1:8181",
		MemberServiceGRPCAddr: "127.0.0.1:9181",
		GatewayUseMemberGRPC:  false,

		MySQLDSN:      "",
		RedisAddr:     "",
		RedisPassword: "",
		RedisDB:       0,

		CORSAllowOrigin: "http://localhost:5173",

		ConsulEnabled:    true,
		ConsulAddress:    "",
		ConsulDatacenter: "dc1",
		ServiceHost:      "127.0.0.1",
	}
}

func loadFile(path string) (*AppConfig, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg AppConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func loadDotEnv(path string) error {
	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return fmt.Errorf("invalid .env line %d", lineNo)
		}
		key = strings.TrimSpace(key)
		if key == "" {
			return fmt.Errorf("invalid .env key at line %d", lineNo)
		}
		if _, exists := os.LookupEnv(key); exists {
			continue
		}
		os.Setenv(key, normalizeDotEnvValue(value))
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return nil
}

func normalizeDotEnvValue(value string) string {
	value = strings.TrimSpace(value)
	if len(value) < 2 {
		return value
	}
	if (strings.HasPrefix(value, `"`) && strings.HasSuffix(value, `"`)) ||
		(strings.HasPrefix(value, `'`) && strings.HasSuffix(value, `'`)) {
		return value[1 : len(value)-1]
	}
	return value
}

func merge(dst, src *AppConfig) {
	if src.APIGatewayPort != "" {
		dst.APIGatewayPort = src.APIGatewayPort
	}
	if src.MemberServicePort != "" {
		dst.MemberServicePort = src.MemberServicePort
	}
	if src.MemberServiceGRPCPort != "" {
		dst.MemberServiceGRPCPort = src.MemberServiceGRPCPort
	}
	if src.MemberServiceURL != "" {
		dst.MemberServiceURL = src.MemberServiceURL
	}
	if src.MemberServiceGRPCAddr != "" {
		dst.MemberServiceGRPCAddr = src.MemberServiceGRPCAddr
	}
	if src.GatewayUseMemberGRPC {
		dst.GatewayUseMemberGRPC = true
	}
	if src.MySQLDSN != "" {
		dst.MySQLDSN = src.MySQLDSN
	}
	if src.RedisAddr != "" {
		dst.RedisAddr = src.RedisAddr
	}
	if src.RedisPassword != "" {
		dst.RedisPassword = src.RedisPassword
	}
	if src.RedisDB != 0 {
		dst.RedisDB = src.RedisDB
	}
	if src.CORSAllowOrigin != "" {
		dst.CORSAllowOrigin = src.CORSAllowOrigin
	}
	if src.ConsulEnabled {
		dst.ConsulEnabled = true
	}
	if src.ConsulAddress != "" {
		dst.ConsulAddress = src.ConsulAddress
	}
	if src.ConsulDatacenter != "" {
		dst.ConsulDatacenter = src.ConsulDatacenter
	}
	if src.ServiceHost != "" {
		dst.ServiceHost = src.ServiceHost
	}
}

func overrideWithEnv(cfg *AppConfig) {
	cfg.APIGatewayPort = getenv("API_GATEWAY_PORT", cfg.APIGatewayPort)
	cfg.MemberServicePort = getenv("MEMBER_SERVICE_PORT", cfg.MemberServicePort)
	cfg.MemberServiceGRPCPort = getenv("MEMBER_SERVICE_GRPC_PORT", cfg.MemberServiceGRPCPort)

	cfg.MemberServiceURL = getenv("MEMBER_SERVICE_URL", cfg.MemberServiceURL)
	cfg.MemberServiceGRPCAddr = getenv("MEMBER_SERVICE_GRPC_ADDR", cfg.MemberServiceGRPCAddr)

	cfg.MySQLDSN = getenv("MYSQL_DSN", cfg.MySQLDSN)
	cfg.RedisAddr = getenv("REDIS_ADDR", cfg.RedisAddr)
	cfg.RedisPassword = getenv("REDIS_PASSWORD", cfg.RedisPassword)
	if raw := os.Getenv("REDIS_DB"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			cfg.RedisDB = parsed
		}
	}

	cfg.CORSAllowOrigin = getenv("CORS_ALLOW_ORIGIN", cfg.CORSAllowOrigin)
	cfg.ConsulAddress = getenv("CONSUL_ADDRESS", cfg.ConsulAddress)
	cfg.ConsulDatacenter = getenv("CONSUL_DATACENTER", cfg.ConsulDatacenter)
	cfg.ServiceHost = getenv("SERVICE_HOST", cfg.ServiceHost)

	if raw := os.Getenv("CONSUL_ENABLED"); raw != "" {
		cfg.ConsulEnabled = toBool(raw, cfg.ConsulEnabled)
	}
	if raw := os.Getenv("GATEWAY_USE_MEMBER_GRPC"); raw != "" {
		cfg.GatewayUseMemberGRPC = toBool(raw, cfg.GatewayUseMemberGRPC)
	}
}

func getenv(key, fallback string) string {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	return v
}

func toBool(raw string, fallback bool) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}
