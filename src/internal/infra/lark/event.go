// 事件解析：把飞书原始事件 JSON 归一化成 dto.Event。
// 回调模式与长连接模式共用这里，保证路由层看到的结构完全一致。
package lark

import (
	"encoding/json"
	"fmt"

	"bokeoncall/internal/model/dto"
)

// 飞书事件类型。
const (
	eventTypeMessageReceive = "im.message.receive_v1"
	eventTypeCardAction     = "card.action.trigger"
)

// ParseEvent 解析事件报文。未识别的事件返回 Kind 为空的事件，由路由层忽略。
func ParseEvent(raw []byte) (*dto.Event, error) {
	var env envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, fmt.Errorf("解析事件报文失败: %w", err)
	}
	event := &dto.Event{EventID: env.Header.EventID, EventType: env.Header.EventType}

	switch env.Header.EventType {
	case eventTypeMessageReceive:
		var rawEvent rawMessageEvent
		if err := json.Unmarshal(env.Event, &rawEvent); err != nil {
			return nil, fmt.Errorf("解析消息事件失败: %w", err)
		}
		event.Kind = dto.EventKindMessage
		event.Message = rawEvent.toDTO(env.Header.EventID)
	case eventTypeCardAction:
		var rawEvent rawCardAction
		if err := json.Unmarshal(env.Event, &rawEvent); err != nil {
			return nil, fmt.Errorf("解析卡片回调失败: %w", err)
		}
		event.Kind = dto.EventKindCardAction
		event.CardAction = rawEvent.toDTO(env.Header.EventID)
	}
	return event, nil
}

// envelope 事件信封（schema 2.0）。
type envelope struct {
	Schema string          `json:"schema"`
	Header eventHeader     `json:"header"`
	Event  json.RawMessage `json:"event"`
}

type eventHeader struct {
	EventID    string `json:"event_id"`
	EventType  string `json:"event_type"`
	Token      string `json:"token"`
	CreateTime string `json:"create_time"`
}

// rawMessageEvent 消息事件，只声明需要的字段。
type rawMessageEvent struct {
	Sender struct {
		SenderID struct {
			OpenID string `json:"open_id"`
		} `json:"sender_id"`
	} `json:"sender"`
	Message struct {
		MessageID   string `json:"message_id"`
		RootID      string `json:"root_id"`
		ParentID    string `json:"parent_id"`
		ThreadID    string `json:"thread_id"`
		ChatID      string `json:"chat_id"`
		ChatType    string `json:"chat_type"`
		MessageType string `json:"message_type"`
		Content     string `json:"content"`
		CreateTime  string `json:"create_time"`
		Mentions    []struct {
			Key  string `json:"key"`
			Name string `json:"name"`
			ID   struct {
				OpenID string `json:"open_id"`
			} `json:"id"`
		} `json:"mentions"`
	} `json:"message"`
}

func (m rawMessageEvent) toDTO(eventID string) *dto.MessageEvent {
	mentions := make([]dto.Mention, 0, len(m.Message.Mentions))
	for _, item := range m.Message.Mentions {
		mentions = append(mentions, dto.Mention{Key: item.Key, OpenID: item.ID.OpenID, Name: item.Name})
	}
	return &dto.MessageEvent{
		EventID:      eventID,
		MessageID:    m.Message.MessageID,
		ChatID:       m.Message.ChatID,
		ChatType:     m.Message.ChatType,
		MessageType:  m.Message.MessageType,
		Content:      m.Message.Content,
		SenderOpenID: m.Sender.SenderID.OpenID,
		ThreadID:     m.Message.ThreadID,
		RootID:       m.Message.RootID,
		ParentID:     m.Message.ParentID,
		Mentions:     mentions,
		CreateTime:   m.Message.CreateTime,
	}
}

// rawCardAction 卡片回调事件。
type rawCardAction struct {
	Operator struct {
		OpenID string `json:"open_id"`
		UserID string `json:"user_id"`
	} `json:"operator"`
	Token  string `json:"token"`
	Action struct {
		Value      dto.CardActionValue `json:"value"`
		Tag        string              `json:"tag"`
		Name       string              `json:"name"`
		Option     string              `json:"option"`
		InputValue string              `json:"input_value"`
		FormValue  map[string]any      `json:"form_value"`
	} `json:"action"`
	Context struct {
		OpenMessageID string `json:"open_message_id"`
		OpenChatID    string `json:"open_chat_id"`
	} `json:"context"`
}

func (c rawCardAction) toDTO(eventID string) *dto.CardActionEvent {
	return &dto.CardActionEvent{
		EventID:        eventID,
		MessageID:      c.Context.OpenMessageID,
		ChatID:         c.Context.OpenChatID,
		OperatorOpenID: c.Operator.OpenID,
		Token:          c.Token,
		Tag:            c.Action.Tag,
		Name:           c.Action.Name,
		Option:         c.Action.Option,
		InputValue:     c.Action.InputValue,
		Value:          c.Action.Value,
		FormValue:      c.Action.FormValue,
	}
}
