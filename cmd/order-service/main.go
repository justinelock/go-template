package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	orderapp "go-template/internal/order/app"
	ordermq "go-template/internal/order/mq"
	"go-template/internal/order/repo"
	httptransport "go-template/internal/order/transport/http"
	"go-template/internal/order/worker"
	"go-template/internal/platform/config"
	"go-template/internal/platform/discovery"
	"go-template/internal/platform/health"
	"go-template/internal/platform/httpserver"
	"go-template/internal/platform/idempotency"
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
		panic(err)
	}
	logging.Init(cfg.LogLevel, cfg.LogFormat)

	// 步骤 2：可选初始化 OpenTelemetry。
	otelShutdown, err := platformotel.Init(context.Background(), "order-service", cfg.OtelExporterOTLPEndpoint)
	if err != nil {
		panic(err)
	}
	defer func() { _ = otelShutdown(context.Background()) }()

	port := cfg.OrderServicePort

	// 步骤 3：可选 Consul 注册。
	var consulClient *discovery.Consul
	if cfg.ConsulEnabled {
		consulClient, err = discovery.NewConsul(cfg.ConsulAddress, cfg.ConsulDatacenter)
		if err != nil {
			panic(err)
		}
		serviceID := fmt.Sprintf("order-service-%d", time.Now().UnixNano())
		if err := consulClient.Register(serviceID, "order-service", cfg.ServiceHost, port, "/healthz"); err != nil {
			panic(err)
		}
		defer func() { _ = consulClient.Deregister(serviceID) }()
	}

	// 步骤 4：打开 MySQL、Redis 与 MQ。
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

	bus, err := mq.New(cfg)
	if err != nil {
		panic(err)
	}
	defer bus.Close()
	slog.Info("mq provider", "provider", cfg.MQProvider)

	prober := health.NewProber(
		health.MySQL(db),
		health.Redis(rdb),
		health.FromPing(func(ctx context.Context) error { return bus.Ping() }),
	)

	// 步骤 5：组装仓储、幂等存储、MQ 发布器与领域服务。
	orderRepo := repo.NewMySQLOrderRepo(db)
	idemStore := idempotency.NewStore(rdb, 24*time.Hour)
	publisher := ordermq.NewEventPublisher(bus)
	svc := orderapp.NewService(orderRepo, idemStore, rdb, publisher)

	// 步骤 6：启动 payment.paid 与 order.settle 消费者。
	if err := worker.StartPaymentPaid(bus, svc); err != nil {
		panic(err)
	}
	if err := worker.StartSettlement(bus, svc); err != nil {
		panic(err)
	}

	// 步骤 7：注册 HTTP 健康检查、指标与业务路由。
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", health.LivenessHandler("order-service"))
	mux.HandleFunc("/readyz", health.ReadinessHandler("order-service", prober))
	mux.Handle("/metrics", metrics.Handler())
	httptransport.NewHandler(svc).RegisterRoutes(mux)

	handler := httpserver.Chain("order-service", mux, cfg.OtelEnabled)
	if err := runtime.RunHTTPServer(":"+port, handler, func(ctx context.Context) {
		_ = bus.Close()
	}); err != nil {
		slog.Error("server stopped", "err", err)
	}
}
