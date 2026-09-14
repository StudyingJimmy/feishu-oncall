// 命令执行：先发一张"仅发起人可见"的表单卡片，用户确认提交后再走各命令的处理逻辑。
package command

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"bokeoncall/internal/infra/lark"
	"bokeoncall/internal/model/dto"
	"bokeoncall/pkg/logx"
)

// Handler 命令处理器。
type Handler struct {
	lark *lark.Client
}

// NewHandler 构造。
func NewHandler(larkClient *lark.Client) *Handler { return &Handler{lark: larkClient} }

// Handle 命令入口：按命令分发。
func (h *Handler) Handle(ctx context.Context, msg *dto.MessageEvent, cmd Command, args string) error {
	switch cmd {
	case Upgrade:
		return h.sendCard(ctx, msg, UpgradeCard(), "升级")
	case Transfer:
		return h.sendCard(ctx, msg, TransferCard(), "转接")
	case Event:
		return h.sendCard(ctx, msg, EventCard(), "升级作战室")
	case Save:
		return h.sendCard(ctx, msg, SaveCard(), "工单挂起")
	case Solve:
		return h.handleSolve(ctx, msg, args)
	default:
		return fmt.Errorf("未知命令: %s", cmd)
	}
}

// sendCard 把命令卡片作为"仅发起人可见"卡片发到群里。
// 注意：话题群不支持仅特定人可见卡片，这类群里会发送失败（日志里能看到原因）。
func (h *Handler) sendCard(ctx context.Context, msg *dto.MessageEvent, card any, name string) error {
	messageID, err := h.lark.SendEphemeralCard(ctx, msg.ChatID, msg.SenderOpenID, card)
	if err != nil {
		return fmt.Errorf("发送%s卡片失败: %w", name, err)
	}
	logx.L().Info("已发送命令卡片",
		zap.String("command", name), zap.String("chat_id", msg.ChatID), zap.String("message_id", messageID))
	return nil
}

// HandleCardAction 命令卡片的交互：确认提交 -> handleConfirmXxx，取消 -> 关闭卡片。
func (h *Handler) HandleCardAction(_ context.Context, action *dto.CardActionEvent) (*dto.CardResponse, error) {
	log := logx.L().With(
		zap.String("operator", action.OperatorOpenID),
		zap.String("action", action.Value.Action),
	)
	switch action.Value.Action {
	case dto.ActionCommandUpgrade:
		return h.handleConfirmUpgrade(log, action)
	case dto.ActionCommandTransfer:
		return h.handleConfirmTransfer(log, action)
	case dto.ActionCommandEvent:
		return h.handleConfirmEvent(log, action)
	case dto.ActionCommandSave:
		return h.handleConfirmSave(log, action)
	case dto.ActionCommandCancel:
		log.Info("已取消命令卡片")
		return toast("info", "已取消"), nil
	default:
		return nil, fmt.Errorf("未知的命令动作: %s", action.Value.Action)
	}
}

// ---------- 各命令"确认提交"的处理入口（逻辑先空着）----------

// handleConfirmUpgrade /upgrade 确认提交：TODO 把工单升级到上一层值班。
func (h *Handler) handleConfirmUpgrade(log *zap.Logger, action *dto.CardActionEvent) (*dto.CardResponse, error) {
	log.Info("TODO 未实现 /upgrade 确认提交", zap.String("reason", formValue(action, "reason")))
	return toast("success", "已提交，正在升级"), nil
}

// handleConfirmTransfer /transfer 确认提交：TODO 把工单转派给目标租户。
func (h *Handler) handleConfirmTransfer(log *zap.Logger, action *dto.CardActionEvent) (*dto.CardResponse, error) {
	log.Info("TODO 未实现 /transfer 确认提交",
		zap.String("tenant", formValue(action, "tenant")),
		zap.String("reason", formValue(action, "reason")))
	return toast("success", "已提交，正在转接"), nil
}

// handleConfirmEvent /event 确认提交：TODO 升级为故障并拉起作战室。
func (h *Handler) handleConfirmEvent(log *zap.Logger, action *dto.CardActionEvent) (*dto.CardResponse, error) {
	log.Info("TODO 未实现 /event 确认提交",
		zap.String("room_name", formValue(action, "room_name")),
		zap.String("project", formValue(action, "project")),
		zap.String("reason", formValue(action, "reason")))
	return toast("success", "已提交，正在拉起作战室"), nil
}

// handleConfirmSave /save 确认提交：TODO 暂时挂起工单。
func (h *Handler) handleConfirmSave(log *zap.Logger, action *dto.CardActionEvent) (*dto.CardResponse, error) {
	log.Info("TODO 未实现 /save 确认提交", zap.String("reason", formValue(action, "reason")))
	return toast("success", "已提交，正在挂起"), nil
}

// handleSolve /solve：暂不做卡片，先留日志。
func (h *Handler) handleSolve(_ context.Context, msg *dto.MessageEvent, args string) error {
	logx.L().Info("TODO 未实现 /solve 标记解决", zap.String("args", args), zap.String("message", msg.Brief()))
	return nil
}

// formValue 取表单字段值（字段名对应卡片里的组件 name）。
func formValue(action *dto.CardActionEvent, key string) string {
	value, _ := action.FormValue[key].(string)
	return value
}

// toast 只弹提示，不改卡片。
func toast(toastType, content string) *dto.CardResponse {
	return &dto.CardResponse{Toast: &dto.CardToast{Type: toastType, Content: content}}
}
