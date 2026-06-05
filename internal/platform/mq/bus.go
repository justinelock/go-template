package mq

import "context"

// Message 平台统一消息体；业务只使用逻辑 Topic，不关心底层 MQ 实现。
type Message struct {
	// Topic 逻辑主题，如 TopicOrderSettle。
	Topic string
	// Key 分区/顺序键（RocketMQ MessageKey；Rabbit 可用于日志）。
	Key string
	// Body 消息体。
	Body []byte
	// Headers 可选元数据（traceId 等）。
	Headers map[string]string
}

// Handler 消费回调；返回 error 时由具体实现决定重试策略。
type Handler func(ctx context.Context, msg Message) error

// Bus 消息总线统一出口：发布、订阅、关闭。
type Bus interface {
	// Publish 向逻辑 Topic 发布消息。
	Publish(ctx context.Context, msg Message) error
	// Subscribe 订阅逻辑 Topic；group 在 RabbitMQ 为 consumer tag，在 RocketMQ 为 ConsumerGroup。
	Subscribe(ctx context.Context, topic, group string, handler Handler) error
	// Close 释放连接与消费者。
	Close() error
	// Ping 用于就绪探针。
	Ping() error
}
