package mq

import (
	"context"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// rabbitBus RabbitMQ 版 Bus 实现。
type rabbitBus struct {
	// 步骤 1：AMQP 连接。
	conn *amqp.Connection
	// 步骤 2：发布/消费 channel。
	channel *amqp.Channel
}

// newRabbitBus 建立连接；autoDeclare 为 true 时声明 exchange/queue/binding。
func newRabbitBus(url string, autoDeclare bool) (Bus, error) {
	// 步骤 1：Dial AMQP。
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}

	// 步骤 2：打开 channel。
	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, err
	}

	bus := &rabbitBus{conn: conn, channel: ch}

	// 步骤 3：dev 默认自动声明已知 Topic 的物理资源。
	if autoDeclare {
		if err := bus.declareKnownTopics(); err != nil {
			_ = bus.Close()
			return nil, err
		}
	}

	return bus, nil
}

// declareKnownTopics 声明 order.settle 等已知队列资源。
func (b *rabbitBus) declareKnownTopics() error {
	// 步骤 1：声明 direct exchange。
	if err := b.channel.ExchangeDeclare(RabbitExchange(), "direct", true, false, false, false, nil); err != nil {
		return err
	}

	// 步骤 2：为每个已注册逻辑 Topic 声明 queue 与 binding。
	for topic := range rabbitMappings {
		queue, err := RabbitQueue(topic)
		if err != nil {
			return err
		}
		routingKey, err := RabbitRoutingKey(topic)
		if err != nil {
			return err
		}
		if _, err := b.channel.QueueDeclare(queue, true, false, false, false, nil); err != nil {
			return err
		}
		if err := b.channel.QueueBind(queue, routingKey, RabbitExchange(), false, nil); err != nil {
			return err
		}
	}
	return nil
}

// Publish 发布到逻辑 Topic 映射的 exchange + routing key。
func (b *rabbitBus) Publish(ctx context.Context, msg Message) error {
	// 步骤 1：解析 routing key。
	routingKey, err := RabbitRoutingKey(msg.Topic)
	if err != nil {
		return err
	}

	// 步骤 2：带 context 发布持久化消息。
	return b.channel.PublishWithContext(ctx, RabbitExchange(), routingKey, false, false, amqp.Publishing{
		ContentType:  "text/plain",
		DeliveryMode: amqp.Persistent,
		Body:         msg.Body,
		Timestamp:    time.Now(),
	})
}

// Subscribe 消费逻辑 Topic 对应 queue；失败 nack 并重入队。
func (b *rabbitBus) Subscribe(ctx context.Context, topic, group string, handler Handler) error {
	// 步骤 1：解析 queue 名。
	queue, err := RabbitQueue(topic)
	if err != nil {
		return err
	}

	// 步骤 2：启动 consumer（group 作为 consumer tag）。
	deliveries, err := b.channel.Consume(queue, group, false, false, false, false, nil)
	if err != nil {
		return err
	}

	// 步骤 3：后台循环处理，尊重 ctx 取消。
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case d, ok := <-deliveries:
				if !ok {
					return
				}
				m := Message{Topic: topic, Body: d.Body}
				if err := handler(ctx, m); err != nil {
					_ = d.Nack(false, true)
					continue
				}
				_ = d.Ack(false)
			}
		}
	}()
	return nil
}

// Close 关闭 channel 与连接。
func (b *rabbitBus) Close() error {
	if b.channel != nil {
		_ = b.channel.Close()
	}
	if b.conn != nil {
		return b.conn.Close()
	}
	return nil
}

// Ping 用于健康检查。
func (b *rabbitBus) Ping() error {
	if b.conn == nil || b.conn.IsClosed() {
		return fmt.Errorf("rabbitmq connection closed")
	}
	return nil
}
