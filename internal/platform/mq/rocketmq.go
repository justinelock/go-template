package mq

import (
	"context"
	"fmt"
	"sync"

	"github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/consumer"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/apache/rocketmq-client-go/v2/producer"
)

// rocketBus RocketMQ 版 Bus 实现。
type rocketBus struct {
	// 步骤 1：NameServer 地址。
	nameSrv string
	// 步骤 2：同步生产者。
	producer rocketmq.Producer
	// 步骤 3：已启动的消费者（Close 时 Shutdown）。
	consumers []rocketmq.PushConsumer
	mu        sync.Mutex
}

// newRocketBus 创建生产者；autoDeclare 依赖 Broker autoCreateTopicEnable（见文档）。
func newRocketBus(nameSrv string, _ bool) (Bus, error) {
	// 步骤 1：创建 Producer。
	p, err := rocketmq.NewProducer(
		producer.WithNameServer([]string{nameSrv}),
		producer.WithRetry(2),
	)
	if err != nil {
		return nil, err
	}
	if err := p.Start(); err != nil {
		return nil, err
	}

	return &rocketBus{nameSrv: nameSrv, producer: p}, nil
}

// Publish 向 RocketMQ Topic 发送消息。
func (b *rocketBus) Publish(ctx context.Context, msg Message) error {
	// 步骤 1：逻辑 Topic -> Rocket Topic。
	topic, err := RocketTopic(msg.Topic)
	if err != nil {
		return err
	}

	// 步骤 2：构造消息并设置 MessageKey。
	rmsg := primitive.NewMessage(topic, msg.Body)
	if msg.Key != "" {
		rmsg.WithKeys([]string{msg.Key})
	}

	// 步骤 3：同步发送。
	_, err = b.producer.SendSync(ctx, rmsg)
	return err
}

// Subscribe 使用 PushConsumer 订阅；group 为 ConsumerGroup。
func (b *rocketBus) Subscribe(ctx context.Context, topic, group string, handler Handler) error {
	// 步骤 1：逻辑 Topic -> Rocket Topic。
	rocketTopic, err := RocketTopic(topic)
	if err != nil {
		return err
	}

	// 步骤 2：创建 PushConsumer。
	c, err := rocketmq.NewPushConsumer(
		consumer.WithNameServer([]string{b.nameSrv}),
		consumer.WithGroupName(group),
		consumer.WithConsumeFromWhere(consumer.ConsumeFromLastOffset),
	)
	if err != nil {
		return err
	}

	// 步骤 3：注册回调。
	err = c.Subscribe(rocketTopic, consumer.MessageSelector{}, func(cctx context.Context, msgs ...*primitive.MessageExt) (consumer.ConsumeResult, error) {
		for _, ext := range msgs {
			m := Message{
				Topic: topic,
				Body:  ext.Body,
			}
			if keys := ext.GetKeys(); keys != "" {
				m.Key = keys
			}
			runCtx := cctx
			if ctx.Err() != nil {
				runCtx = ctx
			}
			if err := handler(runCtx, m); err != nil {
				return consumer.ConsumeRetryLater, err
			}
		}
		return consumer.ConsumeSuccess, nil
	})
	if err != nil {
		return err
	}

	// 步骤 4：启动消费者并保存引用。
	if err := c.Start(); err != nil {
		return err
	}
	b.mu.Lock()
	b.consumers = append(b.consumers, c)
	b.mu.Unlock()
	return nil
}

// Close 关闭所有 consumer 与 producer。
func (b *rocketBus) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	var firstErr error
	for _, c := range b.consumers {
		if err := c.Shutdown(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	b.consumers = nil

	if b.producer != nil {
		if err := b.producer.Shutdown(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if firstErr != nil {
		return firstErr
	}
	return nil
}

// Ping 占位：RocketMQ 无轻量 ping，仅检查 nameSrv 已配置。
func (b *rocketBus) Ping() error {
	if b.nameSrv == "" {
		return fmt.Errorf("rocketmq namesrv not configured")
	}
	return nil
}
