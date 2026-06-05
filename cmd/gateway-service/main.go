package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	memberv1 "go-template/api/gen/member/v1"
	gatewayapp "go-template/internal/gateway/app"
	membergrpc "go-template/internal/gateway/client/membergrpc"
	httptransport "go-template/internal/gateway/transport/http"
	"go-template/internal/platform/config"
	"go-template/internal/platform/discovery"
	"go-template/internal/platform/gatewayresilience"
	"go-template/internal/platform/health"
	"go-template/internal/platform/httpserver"
	"go-template/internal/platform/logging"
	"go-template/internal/platform/metrics"
	platformotel "go-template/internal/platform/otel"
	"go-template/internal/platform/ratelimit"
	"go-template/internal/platform/runtime"
	"go-template/internal/platform/store"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	// 步骤 1：加载配置并初始化结构化日志。
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}
	logging.Init(cfg.LogLevel, cfg.LogFormat)
	port := cfg.GatewayServicePort

	// 步骤 2：可选初始化 OpenTelemetry。
	otelShutdown, err := platformotel.Init(context.Background(), "gateway-service", cfg.OtelExporterOTLPEndpoint)
	if err != nil {
		panic(err)
	}
	defer func() { _ = otelShutdown(context.Background()) }()

	// 步骤 3：可选 Consul 注册。
	var consulClient *discovery.Consul
	if cfg.ConsulEnabled {
		consulClient, err = discovery.NewConsul(cfg.ConsulAddress, cfg.ConsulDatacenter)
		if err != nil {
			panic(err)
		}
		serviceID := fmt.Sprintf("gateway-service-%d", time.Now().UnixNano())
		if err := consulClient.Register(serviceID, "gateway-service", cfg.ServiceHost, port, "/healthz"); err != nil {
			panic(err)
		}
		defer func() { _ = consulClient.Deregister(serviceID) }()
	}

	// 步骤 4.1：可选连接 member gRPC（鉴权加速）。
	var memberGRPCClient memberv1.AuthServiceClient
	if cfg.GatewayUseMemberGRPC {
		conn, dialErr := grpc.NewClient(cfg.MemberServiceGRPCAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if dialErr != nil {
			slog.Warn("init member grpc client failed, fallback to http only", "err", dialErr)
		} else {
			memberGRPCClient = memberv1.NewAuthServiceClient(conn)
			defer func() { _ = conn.Close() }()
		}
	}

	// 步骤 4：组装 HTTP 客户端（超时 + 可选 OTel Transport）。
	timeout := time.Duration(cfg.GatewayProxyTimeoutSec) * time.Second
	httpClient := &http.Client{Timeout: timeout}
	if cfg.OtelEnabled {
		httpClient.Transport = otelhttp.NewTransport(http.DefaultTransport)
	}

	resolve := func(serviceName string, fallback string) string {
		if !cfg.ConsulEnabled || consulClient == nil {
			return fallback
		}
		addr, resolveErr := consulClient.ResolveHealthy(serviceName)
		if resolveErr != nil {
			return fallback
		}
		return addr
	}

	// 步骤 5：可选 Redis 限流器。
	var limiter *ratelimit.Limiter
	if cfg.RateLimitEnabled {
		rdb, redisErr := store.OpenRedis(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)
		if redisErr != nil {
			panic(redisErr)
		}
		defer rdb.Close()
		limiter = ratelimit.New(rdb, cfg.RateLimitRedisPrefix)
	}

	// 步骤 5.1：组装鉴权器、限流/断路器与网关 Handler。
	grpcClient := membergrpc.New(cfg.GatewayUseMemberGRPC, cfg.MemberServiceGRPCAddr, memberGRPCClient)
	authenticator := gatewayapp.NewAuthenticator(httpClient, cfg.MemberServiceURL, resolve, grpcClient)
	handler := httptransport.NewHandler(
		httpClient,
		resolve,
		authenticator,
		cfg.MemberServiceURL,
		cfg.OrderServiceURL,
		cfg.PaymentServiceURL,
		cfg.CORSAllowOrigin,
		limiter,
		cfg.RateLimitEnabled,
		gatewayresilience.NewBreakerPool(),
	)

	// 步骤 6：注册健康检查、指标与业务路由。
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", health.LivenessHandler("gateway-service"))
	mux.HandleFunc("/readyz", health.ReadinessHandler("gateway-service", health.NewProber()))
	mux.Handle("/metrics", metrics.Handler())
	handler.RegisterRoutes(mux)

	// 步骤 7：装配中间件链并启动 HTTP 服务。
	root := httpserver.Chain("gateway-service", handler.BuildServer(mux), cfg.OtelEnabled)
	if err := runtime.RunHTTPServer(":"+port, root); err != nil {
		slog.Error("server stopped", "err", err)
	}
}
