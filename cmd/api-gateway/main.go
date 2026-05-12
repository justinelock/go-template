package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	memberv1 "pricing-assistant/api/gen/member/v1"
	gatewayapp "pricing-assistant/internal/gateway/app"
	membergrpc "pricing-assistant/internal/gateway/client/membergrpc"
	httptransport "pricing-assistant/internal/gateway/transport/http"
	"pricing-assistant/internal/platform/config"
	"pricing-assistant/internal/platform/discovery"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config failed: %v", err)
	}
	port := cfg.APIGatewayPort

	var consulClient *discovery.Consul
	if cfg.ConsulEnabled {
		consulClient, err = discovery.NewConsul(cfg.ConsulAddress, cfg.ConsulDatacenter)
		if err != nil {
			log.Fatalf("init consul failed: %v", err)
		}
		serviceID := fmt.Sprintf("api-gateway-%d", time.Now().UnixNano())
		if err := consulClient.Register(serviceID, "api-gateway", cfg.ServiceHost, port, "/healthz"); err != nil {
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

	log.Printf("api-gateway http listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, handler.BuildServer(mux)))
}
