// Package precheck 预检服务：群内提问 -> 发"仅发起人可见"的卡片 -> 匹配知识库 -> 更新卡片。
//
// 结果卡片的两行按钮，点过之后各自变成一行字（其余内容保持不变）：
//
//	@发起人 <结果内容>
//	[👍 有帮助] [👎 无帮助]            -> 「您已选择 👍 有帮助，感谢您的反馈！」
//	────────────
//	[已解决，无需转人工] [仍需转人工]   -> 「您的问题已确认通过预检解决」/「请填写下方卡片确认入群信息」
//
// 点转人工时：原卡片更新成决策文案，同时**另外发一张建群卡片**（问题描述 + 目标租户 + 提交/取消）。
//
// 实现要点：
//  1. 卡片交互的更新走"回调响应里回传整张卡片"（飞书要求，见 infra/lark/dispatcher.go）；
//  2. 不在服务端存状态：结果内容与当前状态都随按钮 value 一起带回来（dto.CardActionValue）。
package precheck

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"

	"bokeoncall/internal/conf"
	"bokeoncall/internal/infra/lark"
	"bokeoncall/internal/model/dto"
	"bokeoncall/pkg/idgen"
	"bokeoncall/pkg/logx"
	"bokeoncall/pkg/timex"
)

// 决策文案（点过底部按钮后，替换那一行按钮）。
const (
	decisionEscalated = "请填写下方卡片确认入群信息"
	decisionSolved    = "您的问题已确认通过预检解决"
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

	// 先给用户的消息回一个表情（默认 OnIt = 敲键盘），让用户知道机器人在响应了
	if err := s.lark.AddReaction(ctx, msg.MessageID, s.cfg.Precheck.ReactionEmoji); err != nil {
		logx.L().Warn("添加表情回复失败", zap.Error(err))
	}

	ref, err := s.sendSearchingCard(ctx, msg, sessionID)
	if err != nil {
		return err
	}

	hits, err := s.searchKnowledge(ctx, msg.Text())
	if err != nil {
		logx.L().Warn("知识库检索失败，直接给转人工入口", zap.Error(err))
	}

	card := ResultCard(msg.SenderOpenID, sessionID, resultText(hits), "", "")
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
// 返回的卡片会被飞书直接更新到用户点击的那张卡片上（同步返回，3 秒内）。
func (s *Service) HandleCardAction(ctx context.Context, action *dto.CardActionEvent) (*dto.CardResponse, error) {
	log := logx.L().With(
		zap.String("session_id", action.Value.SessionID),
		zap.String("operator", action.OperatorOpenID),
		zap.String("action", action.Value.Action),
	)

	switch action.Value.Action {
	case dto.ActionPrecheckHelpful, dto.ActionPrecheckUseless:
		choice := "👍 有帮助"
		if action.Value.Action == dto.ActionPrecheckUseless {
			choice = "👎 无帮助"
		}
		log.Info("收到反馈", zap.String("choice", choice))
		feedback := fmt.Sprintf("您已选择 %s，感谢您的反馈！", choice)
		return rebuild(action, feedback, action.Value.Decision, "感谢您的反馈"), nil

	case dto.ActionPrecheckSkip, dto.ActionPrecheckEscalate:
		// 原卡片更新成决策文案，同时另发一张建群卡片
		toastMsg, ok := s.sendBuildGroupCard(ctx, log, action)
		if !ok {
			return toast("error", toastMsg), nil
		}
		return rebuild(action, action.Value.Feedback, decisionEscalated, toastMsg), nil

	case dto.ActionPrecheckSolved:
		log.Info("已确认解决，无需转人工")
		return rebuild(action, action.Value.Feedback, decisionSolved, "已记录：无需转人工"), nil

	case dto.ActionPrecheckBuildGroup:
		return s.buildGroup(log, action)

	case dto.ActionPrecheckCancel:
		log.Info("已取消建群")
		return toast("info", "已取消"), nil

	case dto.ActionPrecheckUrgent:
		log.Info("收到紧急介入请求（非工作时间）", zap.String("tenant_id", action.Value.TenantID))
		if action.ChatID == "" {
			return toast("error", "缺少群信息，无法通知值班人员"), nil
		}
		s.startService(ctx, log, action.Value.TenantID, action.ChatID, action.Value.Description)
		card := UrgentRequestedCard(s.workTimeText())
		return cardResponse(card, "已通知值班人员立即介入"), nil

	default:
		return nil, fmt.Errorf("未知的预检动作: %s", action.Value.Action)
	}
}

// rebuild 用按钮 value 里带回来的内容重建结果卡片（换掉对应那一行）。
// 如果 value 里没有结果内容（例如是很早以前发的卡片），只弹提示、不动卡片，避免把内容覆盖掉。
func rebuild(action *dto.CardActionEvent, feedback, decision, toastContent string) *dto.CardResponse {
	if action.Value.Result == "" {
		return toast("success", toastContent)
	}
	card := ResultCard(action.OperatorOpenID, action.Value.SessionID, action.Value.Result, feedback, decision)
	return cardResponse(card, toastContent)
}

// sendBuildGroupCard 转人工：另发一张建群表单卡片（仅发起人可见）。返回提示文案与是否成功。
func (s *Service) sendBuildGroupCard(ctx context.Context, log *zap.Logger, action *dto.CardActionEvent) (string, bool) {
	if action.ChatID == "" || action.OperatorOpenID == "" {
		log.Warn("缺少会话信息，无法发送建群卡片")
		return "无法发送建群卡片，请稍后重试", false
	}
	card := BuildGroupCard(action.OperatorOpenID, action.Value.SessionID)
	if _, err := s.lark.SendEphemeralCard(ctx, action.ChatID, action.OperatorOpenID, card); err != nil {
		log.Warn("发送建群卡片失败", zap.Error(err))
		return "建群卡片发送失败，请稍后重试", false
	}
	log.Info("已发送建群表单卡片")
	return "请填写下方卡片确认入群信息", true
}

// buildGroup 收到建群表单：卡片先变成"已提交，正在建群"，再后台建群，建好后替换成成功卡片。
func (s *Service) buildGroup(log *zap.Logger, action *dto.CardActionEvent) (*dto.CardResponse, error) {
	description, _ := action.FormValue["description"].(string)
	tenant, _ := action.FormValue["tenant"].(string)
	log.Info("收到建群表单", zap.String("description", description), zap.String("tenant", tenant))

	go s.createGroupAsync(log, action, description, tenant)
	return cardResponse(BuildGroupPendingCard(action.OperatorOpenID, "已提交，正在建群…"), "已提交，正在建群…"), nil
}

// createGroupAsync 后台建群，完成后把"正在建群"的卡片替换成结果卡片。
func (s *Service) createGroupAsync(log *zap.Logger, action *dto.CardActionEvent, description, tenant string) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	chatID, err := s.createGroup(ctx, action.OperatorOpenID, description, tenant)
	if err != nil {
		log.Error("建群失败", zap.Error(err))
		s.replaceEphemeralCard(ctx, log, action, BuildGroupPendingCard(action.OperatorOpenID,
			fmt.Sprintf("建群失败：%v\n请重新发起或联系值班同学。", err)))
		return
	}
	log.Info("建群成功", zap.String("chat_id", chatID), zap.String("tenant", tenant))
	s.replaceEphemeralCard(ctx, log, action, BuildGroupDoneCard(action.OperatorOpenID, chatID, description, tenant))

	// 按建群时间决定：工作时间直接拉值班人员，非工作时间先发提示卡等用户决定
	if s.cfg.IsWorkTime(timex.Now()) {
		s.startService(ctx, log, tenant, chatID, description)
		return
	}
	log.Info("当前是非工作时间，发送非工作时间提示卡",
		zap.String("chat_id", chatID), zap.String("worktime", s.workTimeText()))
	s.sendWorkTimeCard(ctx, log, tenant, chatID, description)
}

// createGroup 建服务群。
// TODO(派单): 建群时直接带上有权限的成员（现在 members 传空，靠后面 inviteOnDutyL1 拉人）。
func (s *Service) createGroup(ctx context.Context, ownerOpenID, description, tenant string) (string, error) {
	name := "OnCall-" + shorten(description, 20)
	desc := fmt.Sprintf("问题描述：%s\n目标租户：%s", description, tenant)
	return s.lark.CreateChat(ctx, name, desc, ownerOpenID, nil)
}

// startService 进入服务：把该租户的 L1 值班人员拉进群（`inviteOnDutyL1`），
// 然后发一张公开的"工单处理中"卡片（@值班人员 + 问题描述 + 本群可用命令）。
func (s *Service) startService(ctx context.Context, log *zap.Logger, tenantID, chatID, description string) {
	onDuty := s.inviteOnDutyL1(ctx, tenantID, chatID)
	if _, err := s.lark.SendCardToChat(ctx, chatID, WorkingCard(onDuty, description)); err != nil {
		log.Warn("发送工单处理中卡片失败", zap.Error(err))
	}
}

// sendWorkTimeCard 非工作时间：在群里发一张提示卡（公开），等用户决定是否紧急介入。
func (s *Service) sendWorkTimeCard(ctx context.Context, log *zap.Logger, tenantID, chatID, description string) {
	card := WorkTimeCard(s.workTimeText(), tenantID, description)
	if _, err := s.lark.SendCardToChat(ctx, chatID, card); err != nil {
		log.Warn("发送非工作时间提示卡失败", zap.Error(err))
	}
}

// inviteOnDutyL1 把租户下当前 L1 值班人员拉进群，返回被拉的人（用于在卡片里 @ 他们）。
//
// TODO(未实现): 先接值班数据源（oc_duty_shift / 值班服务）查到 tenant 对应的 L1 值班同学 open_id，
// 再调 s.lark.InviteMembers(chatID, openIDs)。现在直接返回空，卡片里是一段占位 @ 文案。
func (s *Service) inviteOnDutyL1(_ context.Context, tenantID, chatID string) []string {
	logx.L().Info("TODO 未实现：按租户拉 L1 值班人员入群",
		zap.String("tenant_id", tenantID), zap.String("chat_id", chatID))
	return nil
}

// workTimeText 把配置的工作时间渲染成卡片文案。
func (s *Service) workTimeText() string {
	if len(s.cfg.WorkTime.Ranges) == 0 {
		return "每日 09:30-12:30、14:00-18:30"
	}
	return "每日 " + strings.Join(s.cfg.WorkTime.Ranges, "、")
}

// replaceEphemeralCard 删掉旧卡片再发一张新卡片（仅特定人可见的卡片没有更新接口）。
func (s *Service) replaceEphemeralCard(ctx context.Context, log *zap.Logger, action *dto.CardActionEvent, card any) {
	if action.ChatID == "" || action.OperatorOpenID == "" {
		log.Warn("缺少会话信息，无法更新卡片")
		return
	}
	if _, err := s.lark.SendEphemeralCard(ctx, action.ChatID, action.OperatorOpenID, card); err != nil {
		log.Warn("发送新卡片失败", zap.Error(err))
		return
	}
	if err := s.lark.DeleteEphemeralCard(ctx, action.MessageID); err != nil {
		log.Warn("删除旧卡片失败，可能出现两张卡片", zap.Error(err))
	}
}

// cardResponse 组装卡片回调响应：更新卡片 + 提示。
func cardResponse(card any, toastContent string) *dto.CardResponse {
	resp := &dto.CardResponse{Card: card}
	if toastContent != "" {
		resp.Toast = &dto.CardToast{Type: "success", Content: toastContent}
	}
	return resp
}

// toast 只弹提示，不改卡片。
func toast(toastType, content string) *dto.CardResponse {
	return &dto.CardResponse{Toast: &dto.CardToast{Type: toastType, Content: content}}
}

// sendSearchingCard 发"检索中"卡片，返回卡片引用（后续更新用）。
//
// 普通对话群走「仅特定人可见」接口（只有发起人能看到）；
// 话题（thread）内该接口不支持，退化成回复到话题里（所有人可见）。
func (s *Service) sendSearchingCard(ctx context.Context, msg *dto.MessageEvent, sessionID string) (cardRef, error) {
	card := SearchingCard(msg.SenderOpenID, sessionID)
	if msg.ThreadID == "" {
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
	messageID, err := s.lark.ReplyCard(ctx, msg.MessageID, card, lark.ReplyOpts{InThread: true})
	if err != nil {
		return cardRef{}, err
	}
	return cardRef{MessageID: messageID}, nil
}

// replace 用新卡片替换旧卡片（只用于"检索中 -> 结果"这种非交互更新）。
// 仅特定人可见的卡片飞书没有更新接口，只能删掉旧的再发一张新的。
func (s *Service) replace(ctx context.Context, ref cardRef, chatID, openID string, card any) error {
	if !ref.Ephemeral {
		return s.lark.PatchCard(ctx, ref.MessageID, card)
	}
	if chatID == "" || openID == "" {
		return fmt.Errorf("缺少 chat_id 或 open_id，无法更新仅特定人可见卡片")
	}
	// 先发新卡片再删旧卡片：万一发送失败，用户至少还能看到原来那张
	if _, err := s.lark.SendEphemeralCard(ctx, chatID, openID, card); err != nil {
		return err
	}
	if err := s.lark.DeleteEphemeralCard(ctx, ref.MessageID); err != nil {
		logx.L().Warn("删除旧的仅特定人可见卡片失败，可能出现两张卡片", zap.Error(err))
	}
	return nil
}

// searchKnowledge 知识库检索（三种知识源混合召回）。
// TODO: 接 infra/es，做「FAQ + 文档 + 历史工单」的 BM25 与向量混合检索再排序取 TopK。
// 现在先固定等 2 秒返回空结果，用来跑通"检索中 -> 结果"的卡片链路。
func (s *Service) searchKnowledge(_ context.Context, query string) ([]dto.KbHit, error) {
	logx.L().Info("知识库检索（占位实现）：等待 2s 后返回空结果", zap.String("query", query))
	time.Sleep(2 * time.Second)
	return nil, nil
}
