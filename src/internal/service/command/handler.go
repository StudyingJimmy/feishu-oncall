// 命令执行：Handle 按命令分发到 5 个处理函数。
// 每个处理函数是对应的"触发入口"，具体业务逻辑（改工单状态、拉作战室等）在后面填。
package command

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"bokeoncall/internal/model/dto"
	"bokeoncall/pkg/logx"
)

// Handler 命令处理器。
type Handler struct {
	// TODO(依赖注入): ticket service（升级 / 转派 / 解决 / 挂起）、lark client（拉作战室、发消息）
}

// NewHandler 构造。
func NewHandler() *Handler { return &Handler{} }

// Handle 命令入口：按命令分发。
func (h *Handler) Handle(ctx context.Context, msg *dto.MessageEvent, cmd Command, args string) error {
	switch cmd {
	case Upgrade:
		return h.handleUpgrade(ctx, msg, args)
	case Transfer:
		return h.handleTransfer(ctx, msg, args)
	case Event:
		return h.handleEvent(ctx, msg, args)
	case Solve:
		return h.handleSolve(ctx, msg, args)
	case Save:
		return h.handleSave(ctx, msg, args)
	default:
		return fmt.Errorf("未知命令: %s", cmd)
	}
}

// handleUpgrade /upgrade [层级] [原因]：把工单升级到上一层值班。
func (h *Handler) handleUpgrade(_ context.Context, msg *dto.MessageEvent, args string) error {
	logx.L().Info("TODO 未实现 /upgrade 升级", zap.String("args", args), zap.String("message", msg.Brief()))
	return nil
}

// handleTransfer /transfer <团队|人> [原因]：把工单转派给其它团队或同学。
func (h *Handler) handleTransfer(_ context.Context, msg *dto.MessageEvent, args string) error {
	logx.L().Info("TODO 未实现 /transfer 转派", zap.String("args", args), zap.String("message", msg.Brief()))
	return nil
}

// handleEvent /event [说明]：升级为故障，拉起故障作战室（拉群 + 通知相关人）。
func (h *Handler) handleEvent(_ context.Context, msg *dto.MessageEvent, args string) error {
	logx.L().Info("TODO 未实现 /event 故障作战室", zap.String("args", args), zap.String("message", msg.Brief()))
	return nil
}

// handleSolve /solve [结论]：标记工单已解决。
func (h *Handler) handleSolve(_ context.Context, msg *dto.MessageEvent, args string) error {
	logx.L().Info("TODO 未实现 /solve 标记解决", zap.String("args", args), zap.String("message", msg.Brief()))
	return nil
}

// handleSave /save [说明]：暂时挂起工单（保存现场，后续再恢复）。
func (h *Handler) handleSave(_ context.Context, msg *dto.MessageEvent, args string) error {
	logx.L().Info("TODO 未实现 /save 暂时挂起", zap.String("args", args), zap.String("message", msg.Brief()))
	return nil
}
