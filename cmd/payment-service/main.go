package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	paymentapp "go-template/internal/payment/app"
	paymentmq "go-template/internal/payment/mq"
	"go-template/internal/payment/repo"
	httptransport "go-template/internal/payment/transport/http"
	"go-template/internal/payment/worker"
	"go-template/internal/platform/config"
	"go-template/internal/platform/discovery"
	"go-template/internal/platform/health"
	"go-template/internal/platform/httpserver"
	"go-template/internal/platform/logging"
	"go-template/internal/platform/metrics"
	"go-template/internal/platform/mq"
	platformotel "go-template/internal/platform/otel"
	"go-template/internal/platform/runtime"
	"go-template/internal/platform/store"
)

func main() {
	// 步骤 1：加载配置、日志与可选 OTel。
	cfg, err := config.Load()
	if err != nil {
		slog.Error("load config failed", "err", err)
		panic(err)
	}
	logging.Init(cfg.LogLevel, cfg.LogFormat)

	// 步骤 2：可选初始化 OpenTelemetry。
	otelShutdown, err := platformotel.Init(context.Background(), "payment-service", cfg.OtelExporterOTLPEndpoint)
	if err != nil {
		slog.Error("otel init failed", "err", err)
		panic(err)
	}
	defer func() { _ = otelShutdown(context.Background()) }()

	port := cfg.PaymentServicePort
	// 步骤 3：可选 Consul 注册。
	var consulClient *discovery.Consul
	if cfg.ConsulEnabled {
		consulClient, err = discovery.NewConsul(cfg.ConsulAddress, cfg.ConsulDatacenter)
		if err != nil {
			panic(err)
		}
		serviceID := fmt.Sprintf("payment-service-%d", time.Now().UnixNano())
		if err := consulClient.Register(serviceID, "payment-service", cfg.ServiceHost, port, "/healthz"); err != nil {
			panic(err)
		}
		defer func() { _ = consulClient.Deregister(serviceID) }()
	}

	// 步骤 4：打开 MySQL 与 MQ。
	db, err := store.OpenMySQL(cfg.MySQLDSN)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	bus, err := mq.New(cfg)
	if err != nil {
		panic(err)
	}
	defer bus.Close()

	// 步骤 5：组装仓储、MQ 发布器与领域服务，并启动 payment.created 消费者。
	paymentRepo := repo.NewMySQLPaymentRepo(db)
	publisher := paymentmq.NewPublisher(bus)
	svc := paymentapp.NewService(paymentRepo, publisher, cfg.PaymentMockPayEnabled)
	if err := worker.StartPaymentCreated(bus, svc); err != nil {
		panic(err)
	}

	prober := health.NewProber(health.MySQL(db), health.FromPing(func(ctx context.Context) error {
		return bus.Ping()
	}))

	// 步骤 6：注册 HTTP 健康检查、指标与业务路由。
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", health.LivenessHandler("payment-service"))
	mux.HandleFunc("/readyz", health.ReadinessHandler("payment-service", prober))
	mux.Handle("/metrics", metrics.Handler())
	httptransport.NewHandler(svc).RegisterRoutes(mux)

	handler := httpserver.Chain("payment-service", mux, cfg.OtelEnabled)
	if err := runtime.RunHTTPServer(":"+port, handler); err != nil {
		slog.Error("server stopped", "err", err)
	}
}
