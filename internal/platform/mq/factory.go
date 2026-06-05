package mq

import (
	"fmt"
	"strings"

	"go-template/internal/platform/config"
)

// New 按配置 MQ_PROVIDER 创建 Bus；默认 rabbitmq。
func New(cfg *config.AppConfig) (Bus, error) {
	// 步骤 1：解析 provider，空值回退 rabbitmq。
	provider := strings.ToLower(strings.TrimSpace(cfg.MQProvider))
	if provider == "" {
		provider = "rabbitmq"
	}

	// 步骤 2：按 provider 构造实现并校验必填连接参数。
	switch provider {
	case "rabbitmq":
		if strings.TrimSpace(cfg.RabbitMQURL) == "" {
			return nil, fmt.Errorf("rabbitmq_url is required when mq_provider=rabbitmq")
		}
		return newRabbitBus(cfg.RabbitMQURL, cfg.MQAutoDeclare)
	case "rocketmq":
		if strings.TrimSpace(cfg.RocketMQNameSrv) == "" {
			return nil, fmt.Errorf("rocketmq_namesrv is required when mq_provider=rocketmq")
		}
		return newRocketBus(cfg.RocketMQNameSrv, cfg.MQAutoDeclare)
	default:
		return nil, fmt.Errorf("unsupported mq_provider: %s (supported: rabbitmq, rocketmq)", provider)
	}
}
