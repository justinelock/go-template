package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	memberv1 "go-template/api/gen/member/v1"
	memberapp "go-template/internal/member/app"
	"go-template/internal/member/repo"
	grpctransport "go-template/internal/member/transport/grpc"
	httptransport "go-template/internal/member/transport/http"
	"go-template/internal/platform/config"
	"go-template/internal/platform/discovery"
	"go-template/internal/platform/health"
	"go-template/internal/platform/httpserver"
	"go-template/internal/platform/logging"
	"go-template/internal/platform/metrics"
	platformotel "go-template/internal/platform/otel"
	"go-template/internal/platform/runtime"
	"go-template/internal/platform/store"

	"google.golang.org/grpc"
)

func main() {
	// 步骤 1：加载配置、日志与可选 OTel。
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}
	logging.Init(cfg.LogLevel, cfg.LogFormat)

	// 步骤 2：可选初始化 OpenTelemetry。
	otelShutdown, err := platformotel.Init(context.Background(), "member-service", cfg.OtelExporterOTLPEndpoint)
	if err != nil {
		panic(err)
	}
	defer func() { _ = otelShutdown(context.Background()) }()

	port := cfg.MemberServicePort
	// 步骤 3：可选 Consul 注册。
	var consulClient *discovery.Consul
	if cfg.ConsulEnabled {
		consulClient, err = discovery.NewConsul(cfg.ConsulAddress, cfg.ConsulDatacenter)
		if err != nil {
			panic(err)
		}
		serviceID := fmt.Sprintf("member-service-%d", time.Now().UnixNano())
		if err := consulClient.Register(serviceID, "member-service", cfg.ServiceHost, port, "/healthz"); err != nil {
			panic(err)
		}
		defer func() { _ = consulClient.Deregister(serviceID) }()
	}

	// 步骤 4：打开 MySQL 与 Redis 并组装就绪探针。
	db, err := store.OpenMySQL(cfg.MySQLDSN)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	rdb, err := store.OpenRedis(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)
	if err != nil {
		panic(err)
	}
	defer rdb.Close()

	prober := health.NewProber(health.MySQL(db), health.Redis(rdb))

	// 步骤 5：组装仓储与领域服务。
	userRepo := repo.NewMySQLUserRepo(db)
	tokenRepo := repo.NewRedisTokenRepo(rdb)
	svc := memberapp.NewService(userRepo, tokenRepo)

	// 步骤 6：注册 HTTP 健康检查、指标与业务路由。
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", health.LivenessHandler("member-service"))
	mux.HandleFunc("/readyz", health.ReadinessHandler("member-service", prober))
	mux.Handle("/metrics", metrics.Handler())
	httptransport.NewHandler(svc).RegisterRoutes(mux)

	// 步骤 7：并行启动 gRPC AuthService（供网关 introspect）。
	grpcPort := cfg.MemberServiceGRPCPort
	lis, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		panic(err)
	}
	grpcServer := grpc.NewServer()
	memberv1.RegisterAuthServiceServer(grpcServer, grpctransport.NewAuthServer(svc))
	go func() {
		slog.Info("member-service grpc listening", "port", grpcPort)
		if err := grpcServer.Serve(lis); err != nil {
			slog.Error("grpc serve failed", "err", err)
		}
	}()

	handler := httpserver.Chain("member-service", mux, cfg.OtelEnabled)
	if err := runtime.RunHTTPServer(":"+port, handler, func(ctx context.Context) {
		grpcServer.GracefulStop()
	}); err != nil {
		slog.Error("server stopped", "err", err)
	}
}
