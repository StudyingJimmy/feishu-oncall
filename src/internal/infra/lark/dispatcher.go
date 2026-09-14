// 事件分发：SDK 的 EventDispatcher 负责解密、URL 校验（challenge）、token 校验与按类型分发；
// 我们只注册关心的两类事件，并把原始报文交给统一解析（event.go）。
package lark

import (
	"context"
	"encoding/json"
	"time"

	larkevent "github.com/larksuite/oapi-sdk-go/v3/event"
	"github.com/larksuite/oapi-sdk-go/v3/event/dispatcher"
	larkcallback "github.com/larksuite/oapi-sdk-go/v3/event/dispatcher/callback"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
	"go.uber.org/zap"

	"bokeoncall/internal/conf"
	"bokeoncall/internal/model/dto"
	"bokeoncall/pkg/logx"
)

// sinkTimeout 事件交给业务处理后的超时时间。
const sinkTimeout = time.Minute

// newDispatcher 构造事件分发器，回调模式与长连接模式共用。
func newDispatcher(cfg conf.LarkConfig, sink dto.Sink) *dispatcher.EventDispatcher {
	return dispatcher.NewEventDispatcher(cfg.VerificationToken, cfg.EncryptKey).
		OnP2MessageReceiveV1(func(_ context.Context, event *larkim.P2MessageReceiveV1) error {
			return forward(event.EventReq, event, sink)
		}).
		OnP2CardActionTrigger(func(_ context.Context, event *larkcallback.CardActionTriggerEvent) (*larkcallback.CardActionTriggerResponse, error) {
			return nil, forward(event.EventReq, event, sink)
		})
}

// forward 解析事件并异步交给路由。
//
// 之所以异步：飞书要求 3 秒内响应，而预检检索、拉群可能更慢。
// 这里用独立 context —— HTTP 请求的 context 在响应写回后就失效了。
func forward(req *larkevent.EventReq, fallback any, sink dto.Sink) error {
	event, err := ParseEvent(rawBody(req, fallback))
	if err != nil {
		// 脏数据不返回错误，避免飞书反复重推
		logx.L().Warn("解析飞书事件失败", zap.Error(err))
		return nil
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), sinkTimeout)
		defer cancel()
		if err := sink(ctx, event); err != nil {
			logx.L().Error("事件处理失败", zap.String("event_id", event.EventID), zap.Error(err))
		}
	}()
	return nil
}

// rawBody 优先用 SDK 附带的原始报文（保留它没建模的字段，如卡片回调的 context），
// 缺失时退回序列化事件对象。
func rawBody(req *larkevent.EventReq, fallback any) []byte {
	if req != nil && len(req.Body) > 0 {
		return req.Body
	}
	raw, _ := json.Marshal(fallback)
	return raw
}
