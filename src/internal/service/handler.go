// Package service 业务层入口：实现 router.Handler，把事件交给具体业务处理器。
//
// 现状：
//   - 群内提问  -> service/precheck（流式卡片已实现：先发检索中卡片，检索完原地更新）
//   - 群内命令  -> service/command（5 个命令入口）
//   - 单聊提问、工单卡片回调 还是日志占位，等对应 service 实现
package service

import (
	"context"

	"go.uber.org/zap"

	"bokeoncall/internal/model/dto"
	"bokeoncall/internal/service/command"
	"bokeoncall/internal/service/precheck"
	"bokeoncall/pkg/logx"
)

// Handler 事件处理器。
type Handler struct {
	commands *command.Handler
	precheck *precheck.Service
}

// NewHandler 构造。
func NewHandler(precheckSvc *precheck.Service) *Handler {
	return &Handler{commands: command.NewHandler(), precheck: precheckSvc}
}

// HandleQuestionP2P 单聊提问（待实现：私聊里发起预检）。
func (h *Handler) HandleQuestionP2P(_ context.Context, msg *dto.MessageEvent) error {
	logx.L().Info("TODO 未实现 私聊预检", zap.String("message", msg.Brief()))
	return nil
}

// HandleQuestionChat 群内提问：走预检流式卡片。
func (h *Handler) HandleQuestionChat(ctx context.Context, msg *dto.MessageEvent) error {
	return h.precheck.HandleChat(ctx, msg)
}

// HandleCommand 群内命令：由命令处理器按命令分发。
func (h *Handler) HandleCommand(ctx context.Context, msg *dto.MessageEvent, cmd command.Command, args string) error {
	return h.commands.Handle(ctx, msg, cmd, args)
}

// HandleCardAction 卡片回调：预检卡片交预检服务，工单卡片待实现。
func (h *Handler) HandleCardAction(ctx context.Context, action *dto.CardActionEvent) error {
	if dto.IsPrecheckAction(action.Value.Action) {
		return h.precheck.HandleCardAction(ctx, action)
	}
	logx.L().Info("TODO 未实现 工单卡片回调", zap.String("card", action.Brief()))
	return nil
}
