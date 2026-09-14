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
			// 卡片回调必须同步返回：飞书用响应体决定"更新卡片 / 弹提示"，
			// 返回 nil 或空响应都会让客户端提示交互出错。
			resp, err := sink(context.Background(), parseOrLog(event.EventReq, event))
			if err != nil {
				logx.L().Error("卡片回调处理失败", zap.Error(err))
				return &larkcallback.CardActionTriggerResponse{
					Toast: &larkcallback.Toast{Type: "error", Content: "处理失败，请稍后重试"},
				}, nil
			}
			return toCardResponse(resp), nil
		})
}

// forward 解析事件并异步交给路由。
//
// 消息事件可以慢慢处理（飞书只要求 3 秒内响应回调本身），所以异步执行；
// 卡片回调不能异步（见上面的说明），它走 forwardSync。
func forward(req *larkevent.EventReq, fallback any, sink dto.Sink) error {
	event := parseOrLog(req, fallback)
	if event == nil {
		return nil
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), sinkTimeout)
		defer cancel()
		if _, err := sink(ctx, event); err != nil {
			logx.L().Error("事件处理失败", zap.String("event_id", event.EventID), zap.Error(err))
		}
	}()
	return nil
}

// parseOrLog 解析事件；脏数据只告警不报错（避免飞书反复重推）。
func parseOrLog(req *larkevent.EventReq, fallback any) *dto.Event {
	event, err := ParseEvent(rawBody(req, fallback))
	if err != nil {
		logx.L().Warn("解析飞书事件失败", zap.Error(err))
		return nil
	}
	return event
}

// toCardResponse 把内部响应转成 SDK 需要的结构。卡片用 JSON 构建，所以 type 固定为 raw。
func toCardResponse(resp *dto.CardResponse) *larkcallback.CardActionTriggerResponse {
	out := &larkcallback.CardActionTriggerResponse{}
	if resp == nil {
		return out
	}
	if resp.Toast != nil {
		out.Toast = &larkcallback.Toast{Type: resp.Toast.Type, Content: resp.Toast.Content}
	}
	if resp.Card != nil {
		out.Card = &larkcallback.Card{Type: "raw", Data: resp.Card}
	}
	return out
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
