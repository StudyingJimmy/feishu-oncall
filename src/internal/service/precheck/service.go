// Package precheck 预检服务：群内提问 -> 发"仅发起人可见"的卡片 -> 匹配知识库 -> 更新卡片。
//
// 卡片交互流程：
//
//	@发起人 正在匹配知识库...        <- 先发出去，用户立刻有反馈
//	────────────
//	[跳过，转人工]
//
//	检索完成后更新：
//
//	@发起人 <检索结果>
//	[👍 有帮助] [👎 无帮助]
//	────────────
//	[已解决，无需转人工] [仍需转人工]
//
//	点 👍 / 👎  -> 换成「您已选择 xxx，感谢您的反馈！」（仍保留下方两个按钮）
//	点 转人工    -> 换成「建群信息」表单：问题描述 + 目标租户，提交后去建群
package precheck

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"bokeoncall/internal/conf"
	"bokeoncall/internal/infra/lark"
	"bokeoncall/internal/model/dto"
	"bokeoncall/pkg/idgen"
	"bokeoncall/pkg/logx"
	"bokeoncall/pkg/timex"
)

// Service 预检服务。
type Service struct {
	cfg  *conf.Config
	lark *lark.Client
}

// NewService 构造。
func NewService(cfg *conf.Config, larkClient *lark.Client) *Service {
	return &Service{cfg: cfg, lark: larkClient}
}

// cardRef 卡片引用；Ephemeral=true 表示是"仅特定人可见"的卡片。
type cardRef struct {
	MessageID string
	Ephemeral bool
}

// HandleChat 群内提问：先发"检索中"卡片，再查知识库并更新成结果卡片。
func (s *Service) HandleChat(ctx context.Context, msg *dto.MessageEvent) error {
	if msg.SenderOpenID == "" {
		return fmt.Errorf("消息里没有发送者 open_id")
	}
	sessionID := idgen.SessionID(timex.NowCN())

	ref, err := s.sendSearchingCard(ctx, msg, sessionID)
	if err != nil {
		return err
	}

	hits, err := s.searchKnowledge(ctx, msg.Text())
	if err != nil {
		logx.L().Warn("知识库检索失败，直接给转人工入口", zap.Error(err))
	}

	card := ResultCard(msg.SenderOpenID, sessionID, hits, ref.Ephemeral)
	if err := s.replace(ctx, ref, msg.ChatID, msg.SenderOpenID, card); err != nil {
		return fmt.Errorf("更新预检卡片失败: %w", err)
	}
	logx.L().Info("预检卡片已更新",
		zap.String("session_id", sessionID),
		zap.Int("hits", len(hits)),
		zap.String("message_id", ref.MessageID),
	)
	return nil
}

// HandleCardAction 预检卡片上的交互。
func (s *Service) HandleCardAction(ctx context.Context, action *dto.CardActionEvent) error {
	ref := cardRef{MessageID: action.MessageID, Ephemeral: action.Value.Ephemeral}
	log := logx.L().With(
		zap.String("session_id", action.Value.SessionID),
		zap.String("operator", action.OperatorOpenID),
		zap.String("action", action.Value.Action),
	)

	switch action.Value.Action {
	case dto.ActionPrecheckHelpful:
		log.Info("收到反馈：有帮助")
		card := FeedbackCard(action.OperatorOpenID, action.Value.SessionID, "👍 有帮助", ref.Ephemeral)
		return s.replace(ctx, ref, action.ChatID, action.OperatorOpenID, card)

	case dto.ActionPrecheckUseless:
		log.Info("收到反馈：无帮助")
		card := FeedbackCard(action.OperatorOpenID, action.Value.SessionID, "👎 无帮助", ref.Ephemeral)
		return s.replace(ctx, ref, action.ChatID, action.OperatorOpenID, card)

	case dto.ActionPrecheckSkip, dto.ActionPrecheckEscalate:
		log.Info("转人工：发送建群表单卡片")
		card := BuildGroupCard(action.OperatorOpenID, action.Value.SessionID, ref.Ephemeral)
		return s.replace(ctx, ref, action.ChatID, action.OperatorOpenID, card)

	case dto.ActionPrecheckBuildGroup:
		return s.buildGroup(log, action)

	case dto.ActionPrecheckSolved:
		log.Info("TODO 未实现 标记已解决")
		return nil

	default:
		return fmt.Errorf("未知的预检动作: %s", action.Value.Action)
	}
}

// buildGroup 收到建群表单：TODO 建群 + 建单（问题描述与目标租户都在表单里）。
func (s *Service) buildGroup(log *zap.Logger, action *dto.CardActionEvent) error {
	description, _ := action.FormValue["description"].(string)
	tenant, _ := action.FormValue["tenant"].(string)
	log.Info("TODO 未实现 建群建单",
		zap.String("description", description), zap.String("tenant", tenant))
	return nil
}

// sendSearchingCard 发"检索中"卡片，返回卡片引用（后续更新用）。
//
// 普通对话群走「仅特定人可见」接口（只有发起人能看到）；
// 话题（thread）内该接口不支持，退化成回复到话题里（所有人可见）。
func (s *Service) sendSearchingCard(ctx context.Context, msg *dto.MessageEvent, sessionID string) (cardRef, error) {
	if msg.ThreadID == "" {
		card := SearchingCard(msg.SenderOpenID, sessionID, true)
		messageID, err := s.lark.SendEphemeralCard(ctx, msg.ChatID, msg.SenderOpenID, card)
		if err != nil {
			return cardRef{}, err
		}
		logx.L().Info("已发送仅发起人可见卡片",
			zap.String("chat_id", msg.ChatID), zap.String("message_id", messageID))
		return cardRef{MessageID: messageID, Ephemeral: true}, nil
	}

	logx.L().Info("话题内不支持仅发起人可见卡片，改为回复到话题",
		zap.String("chat_id", msg.ChatID), zap.String("thread_id", msg.ThreadID))
	card := SearchingCard(msg.SenderOpenID, sessionID, false)
	messageID, err := s.lark.ReplyCard(ctx, msg.MessageID, card, lark.ReplyOpts{InThread: true})
	if err != nil {
		return cardRef{}, err
	}
	return cardRef{MessageID: messageID}, nil
}

// replace 用新卡片替换旧卡片。
// "仅特定人可见"的卡片飞书没有更新接口，只能删掉旧的再发一张新的。
func (s *Service) replace(ctx context.Context, ref cardRef, chatID, openID string, card any) error {
	if !ref.Ephemeral {
		return s.lark.PatchCard(ctx, ref.MessageID, card)
	}
	if chatID == "" || openID == "" {
		return fmt.Errorf("缺少 chat_id 或 open_id，无法更新仅特定人可见卡片")
	}
	if err := s.lark.DeleteEphemeralCard(ctx, ref.MessageID); err != nil {
		logx.L().Warn("删除旧的仅特定人可见卡片失败，仍尝试发送新卡片", zap.Error(err))
	}
	_, err := s.lark.SendEphemeralCard(ctx, chatID, openID, card)
	return err
}

// searchKnowledge 知识库检索（三种知识源混合召回）。
// TODO: 接 infra/es，做「FAQ + 文档 + 历史工单」的 BM25 与向量混合检索再排序取 TopK。
// 现在先固定等 2 秒返回空结果，用来跑通"检索中 -> 结果"的卡片链路。
func (s *Service) searchKnowledge(_ context.Context, query string) ([]dto.KbHit, error) {
	logx.L().Info("知识库检索（占位实现）：等待 2s 后返回空结果", zap.String("query", query))
	time.Sleep(2 * time.Second)
	return nil, nil
}
