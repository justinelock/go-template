package mq

import "fmt"

// 逻辑 Topic / ConsumerGroup（业务与文档统一使用，勿写死 Rabbit/Rocket 物理名）。
const (
	TopicOrderSettle      = "order.settle"
	GroupOrderSettlement  = "order-settlement"
	TopicPaymentCreated   = "payment.created"
	GroupPaymentCreator   = "payment-creator"
	TopicPaymentPaid      = "payment.paid"
	GroupOrderPaymentPaid = "order-payment-paid"
)

// Rabbit 物理资源（Exchange + Queue + RoutingKey）。
const (
	rabbitDemoExchange = "go_template.demo"
)

// rabbitTopicMapping 逻辑 Topic 到 RabbitMQ queue/routingKey 的映射。
type rabbitTopicMapping struct {
	queue      string
	routingKey string
}

var rabbitMappings = map[string]rabbitTopicMapping{
	TopicOrderSettle:    {queue: "order.settle", routingKey: "order.settle"},
	TopicPaymentCreated: {queue: "payment.created", routingKey: "payment.created"},
	TopicPaymentPaid:    {queue: "payment.paid", routingKey: "payment.paid"},
}

// rocket 逻辑 Topic -> RocketMQ Topic 名（建议下划线）。
var rocketTopicNames = map[string]string{
	TopicOrderSettle:    "order_settle",
	TopicPaymentCreated: "payment_created",
	TopicPaymentPaid:    "payment_paid",
}

// RabbitExchange 返回 RabbitMQ exchange 名。
func RabbitExchange() string {
	return rabbitDemoExchange
}

// RabbitQueue 返回逻辑 Topic 对应的 RabbitMQ queue。
func RabbitQueue(topic string) (string, error) {
	m, ok := rabbitMappings[topic]
	if !ok {
		return "", fmt.Errorf("unknown topic for rabbitmq: %s", topic)
	}
	return m.queue, nil
}

// RabbitRoutingKey 返回逻辑 Topic 对应的 routing key。
func RabbitRoutingKey(topic string) (string, error) {
	m, ok := rabbitMappings[topic]
	if !ok {
		return "", fmt.Errorf("unknown topic for rabbitmq: %s", topic)
	}
	return m.routingKey, nil
}

// RocketTopic 返回逻辑 Topic 对应的 RocketMQ Topic。
func RocketTopic(topic string) (string, error) {
	name, ok := rocketTopicNames[topic]
	if !ok {
		return "", fmt.Errorf("unknown topic for rocketmq: %s", topic)
	}
	return name, nil
}
