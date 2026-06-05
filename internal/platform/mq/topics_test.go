package mq

import "testing"

func TestTopicConstants(t *testing.T) {
	if TopicOrderSettle == "" || GroupOrderSettlement == "" {
		t.Fatal("topic/group constants must not be empty")
	}
}

func TestRabbitMappings_orderSettle(t *testing.T) {
	queue, err := RabbitQueue(TopicOrderSettle)
	if err != nil || queue != "order.settle" {
		t.Fatalf("queue=%q err=%v", queue, err)
	}
	rk, err := RabbitRoutingKey(TopicOrderSettle)
	if err != nil || rk != "order.settle" {
		t.Fatalf("routingKey=%q err=%v", rk, err)
	}
	if RabbitExchange() != "go_template.demo" {
		t.Fatalf("exchange=%s", RabbitExchange())
	}
}

func TestRocketMappings_orderSettle(t *testing.T) {
	topic, err := RocketTopic(TopicOrderSettle)
	if err != nil || topic != "order_settle" {
		t.Fatalf("rocket topic=%q err=%v", topic, err)
	}
}

func TestUnknownTopic(t *testing.T) {
	if _, err := RabbitQueue("unknown.topic"); err == nil {
		t.Fatal("expected error for unknown rabbit topic")
	}
	if _, err := RocketTopic("unknown.topic"); err == nil {
		t.Fatal("expected error for unknown rocket topic")
	}
}
