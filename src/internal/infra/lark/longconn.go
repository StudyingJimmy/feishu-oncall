// 长连接模式：本地调试用，不需要公网入口。
//
// 飞书长连接是 protobuf over websocket，手写不现实，这里用官方 SDK 的 ws 客户端；
// 事件分发复用 dispatcher.go 的分发器，所以行为和回调模式完全一致。
package lark

import (
	"context"

	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	"github.com/larksuite/oapi-sdk-go/v3/event/dispatcher"
	larkws "github.com/larksuite/oapi-sdk-go/v3/ws"
	"go.uber.org/zap"

	"bokeoncall/internal/conf"
	"bokeoncall/internal/model/dto"
	"bokeoncall/pkg/logx"
)

// LongConnClient 长连接客户端。
type LongConnClient struct {
	cfg        conf.LarkConfig
	dispatcher *dispatcher.EventDispatcher
	log        *zap.Logger
}

// NewLongConnClient 构造。
func NewLongConnClient(cfg conf.LarkConfig, sink dto.Sink) *LongConnClient {
	return &LongConnClient{cfg: cfg, dispatcher: newDispatcher(cfg, sink), log: logx.L()}
}

// Start 建立长连接并阻塞消费事件，直到 ctx 取消。
func (c *LongConnClient) Start(ctx context.Context) error {
	c.log.Info("正在建立飞书长连接（本地调试用）", zap.String("app_id", c.cfg.AppID))
	client := larkws.NewClient(c.cfg.AppID, c.cfg.AppSecret,
		larkws.WithEventHandler(c.dispatcher),
		larkws.WithLogLevel(larkcore.LogLevelInfo),
		larkws.WithAutoReconnect(true),
		larkws.WithOnReady(func() { c.log.Info("长连接已就绪，等待飞书推送事件") }),
		larkws.WithOnError(func(err error) { c.log.Warn("长连接异常", zap.Error(err)) }),
	)
	return client.Start(ctx)
}
