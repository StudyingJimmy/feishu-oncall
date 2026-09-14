// Package mq 事件发布/订阅抽象。
//
// 现状：默认实现是 logPublisher —— 把事件写进日志，保证本地零依赖可跑通全流程。
//
// 接 RocketMQ 5.x（compose 里已开 8081 gRPC proxy）时，按下面三步做：
//  1. go get github.com/apache/rocketmq-clients/golang/v5
//  2. 在 conf.RocketMQConfig 里已经备好 Endpoint/AccessKey/SecretKey/ProducerGroup/TopicEvents
//  3. 新增 rocketmq.go 实现下面的 Publisher 接口：
//     p, _ := rmq.NewProducer(&rmq.Config{Endpoint: cfg.Endpoint, Credentials: ..., ...})
//     msg := &rmq.Message{Topic: cfg.TopicEvents, Body: body}
//     msg.SetTag(event.Tag); msg.SetKeys(event.Key)
//     p.Send(ctx, msg)
//     然后改 NewPublisher 里 enable=true 的分支返回它即可，上层代码不用动。
package mq

import (
	"context"
	"encoding/json"
	"time"

	"go.uber.org/zap"

	"bokeoncall/internal/conf"
	"bokeoncall/pkg/logx"
)

// Event 统一事件结构（工单全过程事件流的载体）。
type Event struct {
	Topic      string    `json:"topic"`
	Tag        string    `json:"tag"`
	Key        string    `json:"key"` // 业务主键（工单号 / 会话 ID），用于分区顺序
	OccurredAt time.Time `json:"occurred_at"`
	Payload    any       `json:"payload"`
}

// Publisher 事件发布。
type Publisher interface {
	Publish(ctx context.Context, event Event) error
	Close() error
}

// NewPublisher 按配置返回实现。
func NewPublisher(cfg conf.RocketMQConfig) Publisher {
	// TODO(rocketmq): cfg.Enable 为 true 时返回 RocketMQ 实现（见包注释）。
	return &logPublisher{topic: cfg.TopicEvents}
}

type logPublisher struct {
	topic string
}

func (p *logPublisher) Publish(_ context.Context, event Event) error {
	if event.Topic == "" {
		event.Topic = p.topic
	}
	if event.OccurredAt.IsZero() {
		event.OccurredAt = time.Now().UTC()
	}
	body, err := json.Marshal(event.Payload)
	if err != nil {
		body = []byte("{}")
	}
	logx.L().Info("event published",
		zap.String("topic", event.Topic),
		zap.String("tag", event.Tag),
		zap.String("key", event.Key),
		zap.ByteString("payload", body),
	)
	return nil
}

func (p *logPublisher) Close() error { return nil }

// NoopPublisher 明确不发布事件的实现（测试用）。
func NoopPublisher() Publisher { return &noopPublisher{} }

type noopPublisher struct{}

func (noopPublisher) Publish(context.Context, Event) error { return nil }
func (noopPublisher) Close() error                         { return nil }
