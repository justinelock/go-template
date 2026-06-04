package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	memberv1 "go-template/api/gen/member/v1"
	gatewayapp "go-template/internal/gateway/app"
	membergrpc "go-template/internal/gateway/client/membergrpc"
	httptransport "go-template/internal/gateway/transport/http"
	"go-template/internal/platform/config"
	"go-template/internal/platform/discovery"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config failed: %v", err)
	}
	port := cfg.GatewayServicePort

	var consulClient *discovery.Consul
	if cfg.ConsulEnabled {
		consulClient, err = discovery.NewConsul(cfg.ConsulAddress, cfg.ConsulDatacenter)
		if err != nil {
			log.Fatalf("init consul failed: %v", err)
		}
		serviceID := fmt.Sprintf("gateway-service-%d", time.Now().UnixNano())
		if err := consulClient.Register(serviceID, "gateway-service", cfg.ServiceHost, port, "/healthz"); err != nil {
			log.Fatalf("consul register failed: %v", err)
		}
		defer func() { _ = consulClient.Deregister(serviceID) }()
	}

	var memberGRPCClient memberv1.AuthServiceClient
	if cfg.GatewayUseMemberGRPC {
		conn, dialErr := grpc.NewClient(cfg.MemberServiceGRPCAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if dialErr != nil {
			log.Printf("init member grpc client failed, fallback to http only: %v", dialErr)
		} else {
			memberGRPCClient = memberv1.NewAuthServiceClient(conn)
			defer func() { _ = conn.Close() }()
		}
	}

	httpClient := &http.Client{}
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

	grpcClient := membergrpc.New(cfg.GatewayUseMemberGRPC, cfg.MemberServiceGRPCAddr, memberGRPCClient)
	authenticator := gatewayapp.NewAuthenticator(httpClient, cfg.MemberServiceURL, resolve, grpcClient)
	handler := httptransport.NewHandler(
		httpClient,
		resolve,
		authenticator,
		cfg.MemberServiceURL,
		cfg.CORSAllowOrigin,
	)

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	log.Printf("gateway-service http listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, handler.BuildServer(mux)))
}
