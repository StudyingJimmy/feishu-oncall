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
type Sink func(ctx context.Context, event *Event) error

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
	Action    string            `json:"action"`
	SessionID string            `json:"session_id,omitempty"`
	TicketNo  string            `json:"ticket_no,omitempty"`
	TeamKey   string            `json:"team_key,omitempty"`
	Answer    string            `json:"answer,omitempty"`
	Ephemeral bool              `json:"ephemeral,omitempty"` // 卡片是否"仅特定人可见"（这类卡片只能删掉重发）
	Extra     map[string]string `json:"extra,omitempty"`
}

// 卡片动作（写在按钮 value.action 里）。
const (
	ActionPrecheckSkip       = "precheck_skip"        // 跳过，转人工（检索中）
	ActionPrecheckHelpful    = "precheck_helpful"     // 有帮助
	ActionPrecheckUseless    = "precheck_useless"     // 无帮助
	ActionPrecheckSolved     = "precheck_solved"      // 已解决，无需转人工
	ActionPrecheckEscalate   = "precheck_escalate"    // 仍需转人工
	ActionPrecheckBuildGroup = "precheck_build_group" // 提交建群表单
)

// IsPrecheckAction 是否预检卡片动作。
func IsPrecheckAction(action string) bool { return strings.HasPrefix(action, "precheck_") }

func truncate(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max]) + "..."
}
