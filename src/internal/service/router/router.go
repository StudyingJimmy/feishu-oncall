// Package router 事件路由：把飞书事件分发给业务处理器。
//
// 消息路由规则（都写在 handleMessage 里，一眼能看完）：
//
//	单聊                       -> HandleQuestionP2P
//	来自话题（thread）          -> HandleQuestionChat（@ 不 @ 机器人都一样）
//	群里 @机器人 且在 oncall 群：
//	    带可解析命令            -> HandleCommand（/upgrade /transfer /event /solve /save）
//	    其它                   -> HandleQuestionChat
//	其它情况                    -> HandleQuestionChat
//
// 卡片回调直接交 HandleCardAction，由业务层按 value.action 分发。
package router

import (
	"context"

	"go.uber.org/zap"

	"bokeoncall/internal/model/dto"
	"bokeoncall/internal/service/command"
	"bokeoncall/pkg/logx"
)

// Handler 业务处理器，由 service 层实现（当前用 service.LogHandler 打日志占位）。
type Handler interface {
	HandleQuestionP2P(ctx context.Context, msg *dto.MessageEvent) error
	HandleQuestionChat(ctx context.Context, msg *dto.MessageEvent) error
	HandleCommand(ctx context.Context, msg *dto.MessageEvent, cmd command.Command, args string) error
	HandleCardAction(ctx context.Context, action *dto.CardActionEvent) error
}

// Router 事件路由。
type Router struct {
	botOpenID string
	handler   Handler
	log       *zap.Logger
}

// New 构造。botOpenID 由启动时自动获取（见 cmd/server）。
func New(botOpenID string, handler Handler) *Router {
	return &Router{botOpenID: botOpenID, handler: handler, log: logx.L()}
}

// Handle 事件入口：HTTP 回调与长连接都调它。
func (r *Router) Handle(ctx context.Context, ev *dto.Event) error {
	if ev == nil {
		return nil
	}
	switch ev.Kind {
	case dto.EventKindMessage:
		return r.handleMessage(ctx, ev.Message)
	case dto.EventKindCardAction:
		return r.handleCardAction(ctx, ev.CardAction)
	default:
		r.log.Debug("忽略未识别的事件",
			zap.String("event_id", ev.EventID), zap.String("event_type", ev.EventType))
		return nil
	}
}

// handleMessage 消息路由：按字段直接分流。
func (r *Router) handleMessage(ctx context.Context, msg *dto.MessageEvent) error {
	if msg == nil {
		return nil
	}

	// 判断信号：每个判断独立成函数，以后加维度只改对应的那个函数
	p2p := isP2P(msg)
	inThread := isFromThread(msg)
	mentionBot := isMentionBot(msg, r.botOpenID)
	oncallGroup := isOnCallGroup(msg)

	r.log.Info("收到消息",
		zap.String("event_id", msg.EventID),
		zap.Bool("p2p", p2p),
		zap.Bool("thread", inThread),
		zap.Bool("mention_bot", mentionBot),
		zap.Bool("oncall_group", oncallGroup),
		zap.String("text", msg.Text()),
		zap.String("message", msg.Brief()),
	)

	if p2p {
		return r.handler.HandleQuestionP2P(ctx, msg)
	}
	if inThread {
		return r.handler.HandleQuestionChat(ctx, msg)
	}
	if mentionBot && oncallGroup {
		if cmd, args, ok := command.Parse(msg.Text()); ok {
			r.log.Info("命中命令", zap.String("command", string(cmd)), zap.String("args", args))
			return r.handler.HandleCommand(ctx, msg, cmd, args)
		}
	}
	return r.handler.HandleQuestionChat(ctx, msg)
}

// handleCardAction 卡片回调。
func (r *Router) handleCardAction(ctx context.Context, action *dto.CardActionEvent) error {
	if action == nil {
		return nil
	}
	r.log.Info("收到卡片回调",
		zap.String("event_id", action.EventID),
		zap.String("card", action.Brief()),
	)
	return r.handler.HandleCardAction(ctx, action)
}

// isP2P 是否单聊。
func isP2P(msg *dto.MessageEvent) bool { return msg.ChatType == dto.ChatTypeP2P }

// isFromThread 是否来自话题（thread）。
// 目前只认 thread_id；以后要把 root_id / parent_id 也算作话题消息，改这一个函数即可。
func isFromThread(msg *dto.MessageEvent) bool { return msg.ThreadID != "" }

// isMentionBot 是否 @ 了机器人；没拿到 bot open_id 时退化为「@了任意人」。
func isMentionBot(msg *dto.MessageEvent, botOpenID string) bool {
	if botOpenID == "" {
		return len(msg.Mentions) > 0
	}
	for _, mention := range msg.Mentions {
		if mention.OpenID == botOpenID {
			return true
		}
	}
	return false
}

// isOnCallGroup 是否在 oncall 协作群里。
// TODO: 目前恒为 true；以后接团队群配置 / 工单服务群判断，只改这一个函数。
func isOnCallGroup(_ *dto.MessageEvent) bool { return true }
