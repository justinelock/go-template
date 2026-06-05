package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	orderapp "go-template/internal/order/app"
	ordermq "go-template/internal/order/mq"
	"go-template/internal/order/repo"
	httptransport "go-template/internal/order/transport/http"
	"go-template/internal/order/worker"
	"go-template/internal/platform/config"
	"go-template/internal/platform/discovery"
	"go-template/internal/platform/idempotency"
	"go-template/internal/platform/mq"
	"go-template/internal/platform/store"
)

func main() {
	// 步骤 1：加载配置（.env + app.<env>.json + 环境变量）。
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config failed: %v", err)
	}

	port := cfg.OrderServicePort

	// 步骤 2：可选 Consul 注册，便于网关服务发现。
	var consulClient *discovery.Consul
	if cfg.ConsulEnabled {
		consulClient, err = discovery.NewConsul(cfg.ConsulAddress, cfg.ConsulDatacenter)
		if err != nil {
			log.Fatalf("init consul failed: %v", err)
		}
		serviceID := fmt.Sprintf("order-service-%d", time.Now().UnixNano())
		if err := consulClient.Register(serviceID, "order-service", cfg.ServiceHost, port, "/healthz"); err != nil {
			log.Fatalf("consul register failed: %v", err)
		}
		defer func() { _ = consulClient.Deregister(serviceID) }()
	}

	// 步骤 3：初始化 MySQL（订单持久化）。
	db, err := store.OpenMySQL(cfg.MySQLDSN)
	if err != nil {
		log.Fatalf("mysql init failed: %v", err)
	}
	defer db.Close()

	// 步骤 4：初始化 Redis（幂等键 + 分布式锁）。
	rdb, err := store.OpenRedis(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)
	if err != nil {
		log.Fatalf("redis init failed: %v", err)
	}
	defer rdb.Close()

	// 步骤 5：按 MQ_PROVIDER 创建统一 Bus（默认 rabbitmq）。
	bus, err := mq.New(cfg)
	if err != nil {
		log.Fatalf("mq init failed: %v", err)
	}
	defer bus.Close()
	log.Printf("mq provider=%s", cfg.MQProvider)

	// 步骤 6：组装 repo / 幂等 / app（发布经 SettlePublisher 适配 Bus）。
	orderRepo := repo.NewMySQLOrderRepo(db)
	idemStore := idempotency.NewStore(rdb, 24*time.Hour)
	publisher := ordermq.NewSettlePublisher(bus)
	svc := orderapp.NewService(orderRepo, idemStore, rdb, publisher)

	// 步骤 7：启动结算 worker（订阅 order.settle）。
	if err := worker.StartSettlement(bus, svc); err != nil {
		log.Fatalf("settlement worker failed: %v", err)
	}

	// 步骤 8：注册 HTTP 路由并启动监听。
	mux := http.NewServeMux()
	httptransport.NewHandler(svc).RegisterRoutes(mux)

	log.Printf("order-service http listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
