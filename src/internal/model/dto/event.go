// 入站事件：把飞书的两种事件（消息 / 卡片回调）归一化成下面的结构。
// 无论事件是从 HTTP 回调进来还是从长连接进来，后续路由与业务处理都用同一套代码。
package dto

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// EventKind 入站事件类型。
type EventKind string

const (
	EventKindMessage    EventKind = "message"     // im.message.receive_v1
	EventKindCardAction EventKind = "card_action" // card.action.trigger
)

// 会话类型与消息类型。
const (
	ChatTypeP2P   = "p2p"
	ChatTypeGroup = "group"

	MsgTypeText  = "text"
	MsgTypeImage = "image"
	MsgTypeFile  = "file"
)

// Sink 事件接收方：传输层解析出事件后回调它（router.Router.Handle 就是它的实现）。
// 消息事件返回 nil 响应即可；卡片回调返回卡片响应（飞书用它更新卡片 / 弹提示）。
type Sink func(ctx context.Context, event *Event) (*CardResponse, error)

// CardResponse 卡片回调响应。
// Card 非空时，飞书会把用户刚点的那张卡片更新成 Card 的内容（卡片 JSON，1.0/2.0 要与原卡片一致）。
type CardResponse struct {
	Toast *CardToast
	Card  any
}

// CardToast 客户端提示弹窗。
type CardToast struct {
	Type    string `json:"type"` // info / success / error / warning
	Content string `json:"content"`
}

// Event 归一化后的入站事件。
type Event struct {
	Kind       EventKind
	EventID    string // 飞书事件 ID，可用于幂等去重
	EventType  string // 原始事件类型；未识别时 Kind 为空、这里保留原值
	Message    *MessageEvent
	CardAction *CardActionEvent
}

// MessageEvent 消息事件（只保留路由与业务需要的字段）。
type MessageEvent struct {
	EventID      string
	MessageID    string
	ChatID       string
	ChatType     string // p2p / group
	MessageType  string // text / image / file ...
	Content      string // 原始 content（JSON 字符串）
	SenderOpenID string
	ThreadID     string // 话题 ID，非话题消息为空
	RootID       string // 回复场景的根消息 ID
	ParentID     string // 回复场景的父消息 ID
	Mentions     []Mention
	CreateTime   string
}

// Mention 被 @ 的对象。
type Mention struct {
	Key    string // 正文里的占位符，如 @_user_1
	OpenID string
	Name   string
}

// Text 取文本内容（非文本消息返回空串），并去掉 @机器人 留下的占位符。
func (e *MessageEvent) Text() string {
	if e == nil || e.MessageType != MsgTypeText {
		return ""
	}
	var payload struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal([]byte(e.Content), &payload); err != nil {
		return ""
	}
	text := payload.Text
	for _, m := range e.Mentions {
		if m.Key != "" {
			text = strings.ReplaceAll(text, m.Key, " ")
		}
	}
	return strings.TrimSpace(text)
}

// Brief 日志用的紧凑描述。
func (e *MessageEvent) Brief() string {
	if e == nil {
		return "<nil>"
	}
	return fmt.Sprintf("chat=%s(%s) msg=%s type=%s thread=%t sender=%s text=%q",
		e.ChatID, e.ChatType, e.MessageID, e.MessageType, e.ThreadID != "", e.SenderOpenID, truncate(e.Text(), 40))
}

// CardActionEvent 卡片回调事件。
type CardActionEvent struct {
	EventID        string
	MessageID      string // 卡片消息 ID
	ChatID         string // 卡片所在会话
	OperatorOpenID string
	Token          string
	Tag            string // button / select_static ...
	Name           string
	Option         string
	InputValue     string
	Value          CardActionValue // 按钮上携带的我们自己的载荷
	FormValue      map[string]any  // 表单提交内容
}

// Brief 日志用的紧凑描述。
func (e *CardActionEvent) Brief() string {
	if e == nil {
		return "<nil>"
	}
	return fmt.Sprintf("chat=%s msg=%s operator=%s action=%s ticket=%s",
		e.ChatID, e.MessageID, e.OperatorOpenID, e.Value.Action, e.Value.TicketNo)
}

// CardActionValue 我们放进卡片按钮 value 的载荷；具体动作由业务层解释。
type CardActionValue struct {
	Action      string            `json:"action"`
	SessionID   string            `json:"session_id,omitempty"`
	TicketNo    string            `json:"ticket_no,omitempty"`
	TeamKey     string            `json:"team_key,omitempty"`
	Answer      string            `json:"answer,omitempty"`
	Result      string            `json:"result,omitempty"`      // 反馈按钮带上原卡片的结果内容，点击时用来重建卡片
	Feedback    string            `json:"feedback,omitempty"`    // 已反馈文案（空 = 还没反馈过）
	Decision    string            `json:"decision,omitempty"`    // 已决策文案（空 = 还没决策过）
	TenantID    string            `json:"tenant_id,omitempty"`   // 目标租户（紧急介入时用来拉值班人员）
	Description string            `json:"description,omitempty"` // 问题描述
	Extra       map[string]string `json:"extra,omitempty"`
}

// 卡片动作（写在按钮 value.action 里）。
const (
	ActionPrecheckSkip       = "precheck_skip"        // 跳过，转人工（检索中）
	ActionPrecheckHelpful    = "precheck_helpful"     // 有帮助
	ActionPrecheckUseless    = "precheck_useless"     // 无帮助
	ActionPrecheckSolved     = "precheck_solved"      // 已解决，无需转人工
	ActionPrecheckEscalate   = "precheck_escalate"    // 仍需转人工
	ActionPrecheckBuildGroup = "precheck_build_group" // 提交建群表单
	ActionPrecheckCancel     = "precheck_cancel"      // 取消建群
	ActionPrecheckUrgent     = "precheck_urgent"      // 非工作时间：需要紧急介入
)

// IsPrecheckAction 是否预检卡片动作。
func IsPrecheckAction(action string) bool { return strings.HasPrefix(action, "precheck_") }

// 命令卡片动作（/upgrade /transfer /event /save 的表单卡片）。
const (
	ActionCommandUpgrade  = "command_upgrade"  // 提交升级信息
	ActionCommandTransfer = "command_transfer" // 提交转接信息
	ActionCommandEvent    = "command_event"    // 提交作战室信息
	ActionCommandSave     = "command_save"     // 提交挂起信息
	ActionCommandCancel   = "command_cancel"   // 取消
)

// IsCommandAction 是否命令卡片动作。
func IsCommandAction(action string) bool { return strings.HasPrefix(action, "command_") }

func truncate(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max]) + "..."
}
