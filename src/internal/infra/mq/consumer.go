package mq

import (
	"context"

	"bokeoncall/internal/conf"
	"bokeoncall/pkg/logx"
)

// Handler 事件处理函数。
type Handler func(ctx context.Context, event Event) error

// Consumer 事件消费（worker 进程里跑）。
type Consumer interface {
	Start(ctx context.Context) error
	Close() error
}

// NewConsumer 按配置返回实现。当前是“未启用”版本：阻塞到 ctx 取消并打日志，
// 保证 worker 进程在没接 RocketMQ 时也能正常启动。
func NewConsumer(cfg conf.RocketMQConfig, handler Handler) Consumer {
	// TODO(rocketmq): cfg.Enable 为 true 时返回 RocketMQ 简单消费者实现（见 publisher.go 包注释）。
	return &disabledConsumer{cfg: cfg, handler: handler}
}

type disabledConsumer struct {
	cfg     conf.RocketMQConfig
	handler Handler
}

func (c *disabledConsumer) Start(ctx context.Context) error {
	logx.L().Warn("RocketMQ 未启用（rocketmq.enable=false），worker 只跑定时任务；事件将以日志形式输出")
	<-ctx.Done()
	return nil
}

func (c *disabledConsumer) Close() error { return nil }
