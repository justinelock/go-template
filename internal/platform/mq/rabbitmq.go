package mq

import (
	"context"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	SettleExchange   = "go_template.demo"
	SettleQueue      = "order.settle"
	SettleRoutingKey = "order.settle"
)

// Client RabbitMQ 最小封装：声明队列、发布、消费。
type Client struct {
	// 步骤 1：AMQP 连接。
	conn *amqp.Connection
	// 步骤 2：复用 channel（Demo 单 goroutine 发布/消费）。
	channel *amqp.Channel
}

// Connect 建立连接并声明 demo 用 exchange / queue。
func Connect(url string) (*Client, error) {
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

	// 步骤 3：声明 direct exchange 与 durable queue。
	if err := ch.ExchangeDeclare(SettleExchange, "direct", true, false, false, false, nil); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, err
	}
	if _, err := ch.QueueDeclare(SettleQueue, true, false, false, false, nil); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, err
	}
	if err := ch.QueueBind(SettleQueue, SettleRoutingKey, SettleExchange, false, nil); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, err
	}

	return &Client{conn: conn, channel: ch}, nil
}

// Close 关闭 channel 与连接。
func (c *Client) Close() error {
	// 步骤 1：关闭 channel。
	if c.channel != nil {
		_ = c.channel.Close()
	}
	// 步骤 2：关闭连接。
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// PublishSettle 发布订单结算消息（body 为 order ID 字符串）。
func (c *Client) PublishSettle(ctx context.Context, orderID string) error {
	// 步骤 1：带 context 超时发布到 routing key。
	return c.channel.PublishWithContext(ctx, SettleExchange, SettleRoutingKey, false, false, amqp.Publishing{
		ContentType:  "text/plain",
		DeliveryMode: amqp.Persistent,
		Body:         []byte(orderID),
		Timestamp:    time.Now(),
	})
}

// ConsumeSettle 注册结算队列消费者，handler 返回 error 则 nack 并重入队。
func (c *Client) ConsumeSettle(handler func(orderID string) error) error {
	// 步骤 1：启动 consumer，手动 ack。
	deliveries, err := c.channel.Consume(SettleQueue, "order-settlement", false, false, false, false, nil)
	if err != nil {
		return err
	}

	// 步骤 2：循环处理消息。
	go func() {
		for msg := range deliveries {
			orderID := string(msg.Body)
			if err := handler(orderID); err != nil {
				// 步骤 2.1：处理失败 nack 并重入队。
				_ = msg.Nack(false, true)
				continue
			}
			// 步骤 2.2：处理成功 ack。
			_ = msg.Ack(false)
		}
	}()
	return nil
}

// Ping 用于启动时验证连接可用。
func (c *Client) Ping() error {
	// 步骤 1：检查连接未关闭。
	if c.conn == nil || c.conn.IsClosed() {
		return fmt.Errorf("rabbitmq connection closed")
	}
	return nil
}
