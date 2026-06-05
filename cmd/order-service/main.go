package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	orderapp "go-template/internal/order/app"
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

	// 步骤 5：连接 RabbitMQ 并声明结算队列。
	mqClient, err := mq.Connect(cfg.RabbitMQURL)
	if err != nil {
		log.Fatalf("rabbitmq init failed: %v", err)
	}
	defer mqClient.Close()

	// 步骤 6：组装 repo / 幂等存储 / app 用例层。
	orderRepo := repo.NewMySQLOrderRepo(db)
	idemStore := idempotency.NewStore(rdb, 24*time.Hour)
	svc := orderapp.NewService(orderRepo, idemStore, rdb, mqClient)

	// 步骤 7：启动同进程结算 worker（Demo：消费 order.settle 队列）。
	if err := worker.StartSettlement(mqClient, svc); err != nil {
		log.Fatalf("settlement worker failed: %v", err)
	}

	// 步骤 8：注册 HTTP 路由并启动监听。
	mux := http.NewServeMux()
	httptransport.NewHandler(svc).RegisterRoutes(mux)

	log.Printf("order-service http listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
