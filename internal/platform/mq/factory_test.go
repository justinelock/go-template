package mq

import (
	"strings"
	"testing"

	"go-template/internal/platform/config"
)

func testConfig() *config.AppConfig {
	return &config.AppConfig{
		MQProvider:      "rabbitmq",
		MQAutoDeclare:   true,
		RabbitMQURL:     "amqp://guest:guest@127.0.0.1:5672/",
		RocketMQNameSrv: "127.0.0.1:9876",
	}
}

func TestNew_defaultRabbitmq_requiresURL(t *testing.T) {
	cfg := testConfig()
	cfg.RabbitMQURL = ""
	_, err := New(cfg)
	if err == nil || !strings.Contains(err.Error(), "rabbitmq_url") {
		t.Fatalf("want rabbitmq_url error, got %v", err)
	}
}

func TestNew_rocketmq_requiresNameSrv(t *testing.T) {
	cfg := testConfig()
	cfg.MQProvider = "rocketmq"
	cfg.RocketMQNameSrv = ""
	_, err := New(cfg)
	if err == nil || !strings.Contains(err.Error(), "rocketmq_namesrv") {
		t.Fatalf("want rocketmq_namesrv error, got %v", err)
	}
}

func TestNew_unsupportedProvider(t *testing.T) {
	cfg := testConfig()
	cfg.MQProvider = "kafka"
	_, err := New(cfg)
	if err == nil || !strings.Contains(err.Error(), "unsupported mq_provider") {
		t.Fatalf("want unsupported error, got %v", err)
	}
}
