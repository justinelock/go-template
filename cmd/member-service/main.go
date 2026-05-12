package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"google.golang.org/grpc"
	memberv1 "pricing-assistant/api/gen/member/v1"
	memberapp "pricing-assistant/internal/member/app"
	"pricing-assistant/internal/member/repo"
	grpctransport "pricing-assistant/internal/member/transport/grpc"
	httptransport "pricing-assistant/internal/member/transport/http"
	"pricing-assistant/internal/platform/config"
	"pricing-assistant/internal/platform/discovery"
	"pricing-assistant/internal/platform/store"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config failed: %v", err)
	}

	port := cfg.MemberServicePort
	var consulClient *discovery.Consul
	if cfg.ConsulEnabled {
		consulClient, err = discovery.NewConsul(cfg.ConsulAddress, cfg.ConsulDatacenter)
		if err != nil {
			log.Fatalf("init consul failed: %v", err)
		}
		serviceID := fmt.Sprintf("member-service-%d", time.Now().UnixNano())
		if err := consulClient.Register(serviceID, "member-service", cfg.ServiceHost, port, "/healthz"); err != nil {
			log.Fatalf("consul register failed: %v", err)
		}
		defer func() { _ = consulClient.Deregister(serviceID) }()
	}

	db, err := store.OpenMySQL(cfg.MySQLDSN)
	if err != nil {
		log.Fatalf("mysql init failed: %v", err)
	}
	defer db.Close()

	rdb, err := store.OpenRedis(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)
	if err != nil {
		log.Fatalf("redis init failed: %v", err)
	}
	defer rdb.Close()

	userRepo := repo.NewMySQLUserRepo(db)
	tokenRepo := repo.NewRedisTokenRepo(rdb)
	svc := memberapp.NewService(userRepo, tokenRepo)

	mux := http.NewServeMux()
	httptransport.NewHandler(svc).RegisterRoutes(mux)

	grpcPort := cfg.MemberServiceGRPCPort
	lis, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		log.Fatalf("grpc listen failed: %v", err)
	}
	grpcServer := grpc.NewServer()
	memberv1.RegisterAuthServiceServer(grpcServer, grpctransport.NewAuthServer(svc))
	go func() {
		log.Printf("member-service grpc listening on :%s", grpcPort)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("grpc serve failed: %v", err)
		}
	}()

	log.Printf("member-service http listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
