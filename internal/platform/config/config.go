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

// AppConfig 全服务共享配置（.env / JSON / 环境变量合并加载）。
type AppConfig struct {
	GatewayServicePort    string `json:"gateway_service_port"`
	MemberServicePort     string `json:"member_service_port"`
	MemberServiceGRPCPort string `json:"member_service_grpc_port"`

	MemberServiceURL      string `json:"member_service_url"`
	MemberServiceGRPCAddr string `json:"member_service_grpc_addr"`
	GatewayUseMemberGRPC  bool   `json:"gateway_use_member_grpc"`

	// OrderServicePort order-service HTTP 端口。
	OrderServicePort string `json:"order_service_port"`
	// OrderServiceURL 网关回退 order HTTP 基址。
	OrderServiceURL string `json:"order_service_url"`

	MySQLDSN      string `json:"mysql_dsn"`
	RedisAddr     string `json:"redis_addr"`
	RedisPassword string `json:"redis_password"`
	RedisDB       int    `json:"redis_db"`

	// MQProvider 消息实现：rabbitmq（默认）或 rocketmq。
	MQProvider string `json:"mq_provider"`
	// MQAutoDeclare dev 下自动声明 Rabbit exchange/queue（Rocket 依赖 Broker 自动建 Topic）。
	MQAutoDeclare bool `json:"mq_auto_declare"`
	// RabbitMQURL AMQP 连接串（mq_provider=rabbitmq 时必填）。
	RabbitMQURL string `json:"rabbitmq_url"`
	// RocketMQNameSrv NameServer 地址（mq_provider=rocketmq 时必填）。
	RocketMQNameSrv string `json:"rocketmq_namesrv"`

	CORSAllowOrigin string `json:"cors_allow_origin"`

	LogLevel  string `json:"log_level"`
	LogFormat string `json:"log_format"`

	OtelEnabled              bool   `json:"otel_enabled"`
	OtelExporterOTLPEndpoint string `json:"otel_exporter_otlp_endpoint"`

	RateLimitEnabled     bool   `json:"rate_limit_enabled"`
	RateLimitRedisPrefix string `json:"rate_limit_redis_prefix"`

	GatewayProxyTimeoutSec int `json:"gateway_proxy_timeout_sec"`

	// RoutesConfigPath 网关路由表文件路径（留空则仅用代码内置默认路由）。
	RoutesConfigPath string `json:"routes_config_path"`
	// RoutesReloadSec 路由文件热加载轮询间隔秒数（<=0 时仅靠 SIGHUP 触发）。
	RoutesReloadSec int `json:"routes_reload_sec"`

	PaymentServicePort    string `json:"payment_service_port"`
	PaymentServiceURL     string `json:"payment_service_url"`
	PaymentMockPayEnabled bool   `json:"payment_mock_pay_enabled"`

	ConsulEnabled    bool   `json:"consul_enabled"`
	ConsulAddress    string `json:"consul_address"`
	ConsulDatacenter string `json:"consul_datacenter"`
	ServiceHost      string `json:"service_host"`
}

// Load 按优先级合并默认配置、.env、JSON 文件与环境变量。
func Load() (*AppConfig, error) {
	// 步骤 1：从内置默认值起步。
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
		GatewayServicePort:    "8180",
		MemberServicePort:     "8181",
		MemberServiceGRPCPort: "9181",

		MemberServiceURL:      "http://127.0.0.1:8181",
		MemberServiceGRPCAddr: "127.0.0.1:9181",
		GatewayUseMemberGRPC:  false,

		OrderServicePort: "8182",
		OrderServiceURL:  "http://127.0.0.1:8182",

		MySQLDSN:      "",
		RedisAddr:     "",
		RedisPassword: "",
		RedisDB:       0,

		MQProvider:      "rabbitmq",
		MQAutoDeclare:   true,
		RabbitMQURL:     "amqp://guest:guest@127.0.0.1:5672/",
		RocketMQNameSrv: "127.0.0.1:9876",

		CORSAllowOrigin: "http://localhost:5173",

		LogLevel:  "info",
		LogFormat: "json",

		OtelEnabled:              false,
		OtelExporterOTLPEndpoint: "",

		RateLimitEnabled:     false,
		RateLimitRedisPrefix: "ratelimit",

		GatewayProxyTimeoutSec: 5,

		RoutesConfigPath: "configs/routes.json",
		RoutesReloadSec:  10,

		PaymentServicePort:    "8183",
		PaymentServiceURL:     "http://127.0.0.1:8183",
		PaymentMockPayEnabled: true,

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
	if src.GatewayServicePort != "" {
		dst.GatewayServicePort = src.GatewayServicePort
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
	if src.OrderServicePort != "" {
		dst.OrderServicePort = src.OrderServicePort
	}
	if src.OrderServiceURL != "" {
		dst.OrderServiceURL = src.OrderServiceURL
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
	if src.MQProvider != "" {
		dst.MQProvider = src.MQProvider
		// 配置文件显式声明 mq_provider 时，同步采纳 mq_auto_declare（含 false）。
		dst.MQAutoDeclare = src.MQAutoDeclare
	}
	if src.RabbitMQURL != "" {
		dst.RabbitMQURL = src.RabbitMQURL
	}
	if src.RocketMQNameSrv != "" {
		dst.RocketMQNameSrv = src.RocketMQNameSrv
	}
	if src.CORSAllowOrigin != "" {
		dst.CORSAllowOrigin = src.CORSAllowOrigin
	}
	if src.LogLevel != "" {
		dst.LogLevel = src.LogLevel
	}
	if src.LogFormat != "" {
		dst.LogFormat = src.LogFormat
	}
	if src.OtelExporterOTLPEndpoint != "" {
		dst.OtelExporterOTLPEndpoint = src.OtelExporterOTLPEndpoint
	}
	if src.RateLimitRedisPrefix != "" {
		dst.RateLimitRedisPrefix = src.RateLimitRedisPrefix
	}
	if src.PaymentServicePort != "" {
		dst.PaymentServicePort = src.PaymentServicePort
	}
	if src.PaymentServiceURL != "" {
		dst.PaymentServiceURL = src.PaymentServiceURL
	}
	if src.OtelEnabled {
		dst.OtelEnabled = true
	}
	if src.RateLimitEnabled {
		dst.RateLimitEnabled = true
	}
	if src.PaymentMockPayEnabled {
		dst.PaymentMockPayEnabled = true
	}
	if src.GatewayProxyTimeoutSec > 0 {
		dst.GatewayProxyTimeoutSec = src.GatewayProxyTimeoutSec
	}
	if src.RoutesConfigPath != "" {
		dst.RoutesConfigPath = src.RoutesConfigPath
	}
	if src.RoutesReloadSec > 0 {
		dst.RoutesReloadSec = src.RoutesReloadSec
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
	cfg.GatewayServicePort = getenv("GATEWAY_SERVICE_PORT", cfg.GatewayServicePort)
	cfg.MemberServicePort = getenv("MEMBER_SERVICE_PORT", cfg.MemberServicePort)
	cfg.MemberServiceGRPCPort = getenv("MEMBER_SERVICE_GRPC_PORT", cfg.MemberServiceGRPCPort)

	cfg.MemberServiceURL = getenv("MEMBER_SERVICE_URL", cfg.MemberServiceURL)
	cfg.MemberServiceGRPCAddr = getenv("MEMBER_SERVICE_GRPC_ADDR", cfg.MemberServiceGRPCAddr)

	cfg.OrderServicePort = getenv("ORDER_SERVICE_PORT", cfg.OrderServicePort)
	cfg.OrderServiceURL = getenv("ORDER_SERVICE_URL", cfg.OrderServiceURL)

	cfg.MySQLDSN = getenv("MYSQL_DSN", cfg.MySQLDSN)
	cfg.RedisAddr = getenv("REDIS_ADDR", cfg.RedisAddr)
	cfg.RedisPassword = getenv("REDIS_PASSWORD", cfg.RedisPassword)
	if raw := os.Getenv("REDIS_DB"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			cfg.RedisDB = parsed
		}
	}

	cfg.MQProvider = getenv("MQ_PROVIDER", cfg.MQProvider)
	cfg.RabbitMQURL = getenv("RABBITMQ_URL", cfg.RabbitMQURL)
	cfg.RocketMQNameSrv = getenv("ROCKETMQ_NAMESRV", cfg.RocketMQNameSrv)

	if raw := os.Getenv("MQ_AUTO_DECLARE"); raw != "" {
		cfg.MQAutoDeclare = toBool(raw, cfg.MQAutoDeclare)
	}

	cfg.CORSAllowOrigin = getenv("CORS_ALLOW_ORIGIN", cfg.CORSAllowOrigin)
	cfg.LogLevel = getenv("LOG_LEVEL", cfg.LogLevel)
	cfg.LogFormat = getenv("LOG_FORMAT", cfg.LogFormat)
	cfg.OtelExporterOTLPEndpoint = getenv("OTEL_EXPORTER_OTLP_ENDPOINT", cfg.OtelExporterOTLPEndpoint)
	cfg.RateLimitRedisPrefix = getenv("RATE_LIMIT_REDIS_PREFIX", cfg.RateLimitRedisPrefix)

	cfg.PaymentServicePort = getenv("PAYMENT_SERVICE_PORT", cfg.PaymentServicePort)
	cfg.PaymentServiceURL = getenv("PAYMENT_SERVICE_URL", cfg.PaymentServiceURL)

	if raw := os.Getenv("GATEWAY_PROXY_TIMEOUT_SEC"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			cfg.GatewayProxyTimeoutSec = parsed
		}
	}

	cfg.RoutesConfigPath = getenv("ROUTES_CONFIG_PATH", cfg.RoutesConfigPath)
	if raw := os.Getenv("ROUTES_RELOAD_SEC"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			cfg.RoutesReloadSec = parsed
		}
	}

	if raw := os.Getenv("OTEL_ENABLED"); raw != "" {
		cfg.OtelEnabled = toBool(raw, cfg.OtelEnabled)
	}
	if raw := os.Getenv("RATE_LIMIT_ENABLED"); raw != "" {
		cfg.RateLimitEnabled = toBool(raw, cfg.RateLimitEnabled)
	}
	if raw := os.Getenv("PAYMENT_MOCK_PAY_ENABLED"); raw != "" {
		cfg.PaymentMockPayEnabled = toBool(raw, cfg.PaymentMockPayEnabled)
	}

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
