// Package lark 飞书开放平台接入，按职责拆分：
//
//	client.go      主动调接口：发消息、拉群、查用户、下载消息资源（统一用官方 SDK）
//	dispatcher.go  事件分发：注册消息 / 卡片回调，SDK 负责解密、URL 校验、token 校验
//	callback.go    回调模式（HTTP，生产用）
//	longconn.go    长连接模式（WebSocket，本地调试用）
//	event.go       原始事件报文 → dto.Event 归一化
package lark

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	lark "github.com/larksuite/oapi-sdk-go/v3"
	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	larkcontact "github.com/larksuite/oapi-sdk-go/v3/service/contact/v3"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"

	"bokeoncall/internal/conf"
)

// 消息类型。
const (
	msgTypeText = "text"
	msgTypeCard = "interactive"
)

// Message 群消息（群历史消息拉取结果，供业务归档）。
type Message struct {
	MessageID string
	ChatID    string
	Sender    string
	MsgType   string
	Content   string
	CreateAt  string
}

// Client 飞书接口客户端。token 刷新、重试、请求日志都由 SDK 负责。
type Client struct {
	cfg       conf.LarkConfig
	sdk       *lark.Client
	botOpenID string // 自动获取后缓存
}

// NewClient 构造（不发起网络请求）。
func NewClient(cfg conf.LarkConfig) *Client {
	options := []lark.ClientOptionFunc{lark.WithLogLevel(larkcore.LogLevelInfo)}
	if cfg.APIBase != "" {
		options = append(options, lark.WithOpenBaseUrl(cfg.APIBase))
	}
	if cfg.TimeoutSeconds > 0 {
		options = append(options, lark.WithReqTimeout(time.Duration(cfg.TimeoutSeconds)*time.Second))
	}
	return &Client{cfg: cfg, sdk: lark.NewClient(cfg.AppID, cfg.AppSecret, options...)}
}

// Enabled 是否配置了应用凭证。
func (c *Client) Enabled() bool { return c.cfg.Enabled() }

// BotOpenID 机器人 open_id：优先用配置值，其次用自动获取的值。
func (c *Client) BotOpenID() string {
	if c.cfg.BotOpenID != "" {
		return c.cfg.BotOpenID
	}
	return c.botOpenID
}

// EnsureBotOpenID 用 app_id / app_secret 获取机器人 open_id（群里判断「是否 @机器人」要用）。
// 只需调一次，结果缓存在实例里。
func (c *Client) EnsureBotOpenID(ctx context.Context) (string, error) {
	if c.BotOpenID() != "" {
		return c.BotOpenID(), nil
	}
	resp, err := c.sdk.Get(ctx, "/open-apis/bot/v3/info", nil, larkcore.AccessTokenTypeTenant)
	if err != nil {
		return "", fmt.Errorf("请求机器人信息失败: %w", err)
	}
	var payload struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Bot  struct {
			OpenID  string `json:"open_id"`
			AppName string `json:"app_name"`
		} `json:"bot"`
	}
	if err := json.Unmarshal(resp.RawBody, &payload); err != nil {
		return "", fmt.Errorf("解析机器人信息失败: %w", err)
	}
	if payload.Code != 0 {
		return "", fmt.Errorf("获取机器人信息失败: code=%d msg=%s", payload.Code, payload.Msg)
	}
	if payload.Bot.OpenID == "" {
		return "", fmt.Errorf("机器人信息里没有 open_id（应用可能未开启机器人能力）")
	}
	c.botOpenID = payload.Bot.OpenID
	return c.botOpenID, nil
}

// SendText 发文本消息。receiveIDType：open_id / chat_id / user_id / email。
func (c *Client) SendText(ctx context.Context, receiveID, receiveIDType, text string) (string, error) {
	content, err := json.Marshal(map[string]string{"text": text})
	if err != nil {
		return "", err
	}
	return c.send(ctx, receiveID, receiveIDType, msgTypeText, string(content))
}

// SendCard 发卡片消息。
func (c *Client) SendCard(ctx context.Context, receiveID, receiveIDType string, card any) (string, error) {
	content, err := json.Marshal(card)
	if err != nil {
		return "", err
	}
	return c.send(ctx, receiveID, receiveIDType, msgTypeCard, string(content))
}

// SendCardToChat 往群里发卡片，返回消息 ID。
func (c *Client) SendCardToChat(ctx context.Context, chatID string, card any) (string, error) {
	return c.SendCard(ctx, chatID, "chat_id", card)
}

func (c *Client) send(ctx context.Context, receiveID, receiveIDType, msgType, content string) (string, error) {
	req := larkim.NewCreateMessageReqBuilder().
		ReceiveIdType(receiveIDType).
		Body(larkim.NewCreateMessageReqBodyBuilder().
			ReceiveId(receiveID).
			MsgType(msgType).
			Content(content).
			Build()).
		Build()
	resp, err := c.sdk.Im.V1.Message.Create(ctx, req)
	if err != nil {
		return "", fmt.Errorf("发送消息失败: %w", err)
	}
	if !resp.Success() {
		return "", fmt.Errorf("发送消息失败: code=%d msg=%s", resp.Code, resp.Msg)
	}
	return deref(resp.Data.MessageId), nil
}

// ReplyOpts 回复选项（零值 = 普通回复）。
type ReplyOpts struct {
	InThread bool // 回复到话题（reply_in_thread），话题内回复用
}

// ReplyText 在指定消息下回复文本。
func (c *Client) ReplyText(ctx context.Context, messageID, text string, opts ReplyOpts) (string, error) {
	content, err := json.Marshal(map[string]string{"text": text})
	if err != nil {
		return "", err
	}
	return c.reply(ctx, messageID, msgTypeText, string(content), opts)
}

// ReplyCard 在指定消息下回复卡片。
func (c *Client) ReplyCard(ctx context.Context, messageID string, card any, opts ReplyOpts) (string, error) {
	content, err := json.Marshal(card)
	if err != nil {
		return "", err
	}
	return c.reply(ctx, messageID, msgTypeCard, string(content), opts)
}

func (c *Client) reply(ctx context.Context, messageID, msgType, content string, opts ReplyOpts) (string, error) {
	body := larkim.NewReplyMessageReqBodyBuilder().
		MsgType(msgType).
		Content(content)
	if opts.InThread {
		body = body.ReplyInThread(true)
	}
	req := larkim.NewReplyMessageReqBuilder().
		MessageId(messageID).
		Body(body.Build()).
		Build()
	resp, err := c.sdk.Im.V1.Message.Reply(ctx, req)
	if err != nil {
		return "", fmt.Errorf("回复消息失败: %w", err)
	}
	if !resp.Success() {
		return "", fmt.Errorf("回复消息失败: code=%d msg=%s", resp.Code, resp.Msg)
	}
	return deref(resp.Data.MessageId), nil
}

// SendEphemeralCard 发送"仅特定人可见"的卡片：群里只有指定用户能看到，其他成员看不到。
//
// 飞书官方限制：
//   - 只支持普通对话群，**不支持话题群**；
//   - 被指定的用户不会收到消息通知，且只有在线时可见；
//   - 卡片里 @ 人不会触发提及通知。
//
// 接口：POST /open-apis/ephemeral/v1/send
func (c *Client) SendEphemeralCard(ctx context.Context, chatID, openID string, card any) (string, error) {
	body := map[string]any{
		"chat_id":  chatID,
		"open_id":  openID,
		"msg_type": msgTypeCard,
		"card":     card,
	}
	var out struct {
		MessageID string `json:"message_id"`
	}
	if err := c.postData(ctx, "/open-apis/ephemeral/v1/send", body, &out); err != nil {
		return "", err
	}
	if out.MessageID == "" {
		return "", fmt.Errorf("飞书未返回仅特定人可见卡片的 message_id")
	}
	return out.MessageID, nil
}

// DeleteEphemeralCard 删除"仅特定人可见"卡片。
// 飞书没有提供这类卡片的更新接口（只有 send / delete），所以"更新"要删除后重发。
func (c *Client) DeleteEphemeralCard(ctx context.Context, messageID string) error {
	return c.postData(ctx, "/open-apis/ephemeral/v1/delete", map[string]any{"message_id": messageID}, nil)
}

// AddReaction 给消息加一个表情回复（表示"收到了、在处理"）。
// emojiType 取飞书表情文案里的值，见 /document/server-docs/im-v1/message-reaction/emojis-introduce，
// 例如 OnIt（敲键盘）、DONE、THUMBSUP。
func (c *Client) AddReaction(ctx context.Context, messageID, emojiType string) error {
	req := larkim.NewCreateMessageReactionReqBuilder().
		MessageId(messageID).
		Body(larkim.NewCreateMessageReactionReqBodyBuilder().
			ReactionType(larkim.NewEmojiBuilder().EmojiType(emojiType).Build()).
			Build()).
		Build()
	resp, err := c.sdk.Im.V1.MessageReaction.Create(ctx, req)
	if err != nil {
		return fmt.Errorf("添加表情回复失败: %w", err)
	}
	if !resp.Success() {
		return fmt.Errorf("添加表情回复失败: code=%d msg=%s", resp.Code, resp.Msg)
	}
	return nil
}

// postData POST 请求并解析 data 段（SDK 没封装的接口走这里）。
func (c *Client) postData(ctx context.Context, path string, body, out any) error {
	resp, err := c.sdk.Post(ctx, path, body, larkcore.AccessTokenTypeTenant)
	if err != nil {
		return fmt.Errorf("请求飞书失败 %s: %w", path, err)
	}
	var payload struct {
		Code int             `json:"code"`
		Msg  string          `json:"msg"`
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(resp.RawBody, &payload); err != nil {
		return fmt.Errorf("解析飞书响应失败 %s: %w", path, err)
	}
	if payload.Code != 0 {
		return fmt.Errorf("飞书接口返回错误 %s: code=%d msg=%s", path, payload.Code, payload.Msg)
	}
	if out != nil && len(payload.Data) > 0 {
		if err := json.Unmarshal(payload.Data, out); err != nil {
			return fmt.Errorf("解析 data 失败 %s: %w", path, err)
		}
	}
	return nil
}

// PatchCard 原地更新卡片（预检步骤切换、工单状态变化用）。
func (c *Client) PatchCard(ctx context.Context, messageID string, card any) error {
	content, err := json.Marshal(card)
	if err != nil {
		return err
	}
	req := larkim.NewPatchMessageReqBuilder().
		MessageId(messageID).
		Body(larkim.NewPatchMessageReqBodyBuilder().Content(string(content)).Build()).
		Build()
	resp, err := c.sdk.Im.V1.Message.Patch(ctx, req)
	if err != nil {
		return fmt.Errorf("更新卡片失败: %w", err)
	}
	if !resp.Success() {
		return fmt.Errorf("更新卡片失败: code=%d msg=%s", resp.Code, resp.Msg)
	}
	return nil
}

// CreateChat 拉服务群，返回 chat_id。
func (c *Client) CreateChat(ctx context.Context, name, description, ownerOpenID string, memberOpenIDs []string) (string, error) {
	body := larkim.NewCreateChatReqBodyBuilder().
		Name(name).
		Description(description).
		ChatMode("group").
		ChatType("private").
		UserIdList(memberOpenIDs)
	if ownerOpenID != "" {
		body = body.OwnerId(ownerOpenID)
	}
	req := larkim.NewCreateChatReqBuilder().
		UserIdType("open_id").
		Body(body.Build()).
		Build()
	resp, err := c.sdk.Im.V1.Chat.Create(ctx, req)
	if err != nil {
		return "", fmt.Errorf("创建群失败: %w", err)
	}
	if !resp.Success() {
		return "", fmt.Errorf("创建群失败: code=%d msg=%s", resp.Code, resp.Msg)
	}
	return deref(resp.Data.ChatId), nil
}

// InviteMembers 拉人进群。
func (c *Client) InviteMembers(ctx context.Context, chatID string, openIDs []string) error {
	if len(openIDs) == 0 {
		return nil
	}
	req := larkim.NewCreateChatMembersReqBuilder().
		ChatId(chatID).
		MemberIdType("open_id").
		Body(larkim.NewCreateChatMembersReqBodyBuilder().IdList(openIDs).Build()).
		Build()
	resp, err := c.sdk.Im.V1.ChatMembers.Create(ctx, req)
	if err != nil {
		return fmt.Errorf("邀请成员入群失败: %w", err)
	}
	if !resp.Success() {
		return fmt.Errorf("邀请成员入群失败: code=%d msg=%s", resp.Code, resp.Msg)
	}
	return nil
}

// GetUserName 查用户姓名（卡片展示用）。
func (c *Client) GetUserName(ctx context.Context, openID string) (string, error) {
	req := larkcontact.NewGetUserReqBuilder().
		UserId(openID).
		UserIdType("open_id").
		Build()
	resp, err := c.sdk.Contact.V3.User.Get(ctx, req)
	if err != nil {
		return "", fmt.Errorf("查询用户失败: %w", err)
	}
	if !resp.Success() {
		return "", fmt.Errorf("查询用户失败: code=%d msg=%s", resp.Code, resp.Msg)
	}
	if resp.Data == nil || resp.Data.User == nil {
		return "", nil
	}
	return deref(resp.Data.User.Name), nil
}

// ListChatMessages 拉某个时间段内的群消息（结案归档 / 全过程回捞）。
func (c *Client) ListChatMessages(ctx context.Context, chatID string, start, end time.Time) ([]Message, error) {
	req := larkim.NewListMessageReqBuilder().
		ContainerIdType("chat").
		ContainerId(chatID).
		StartTime(fmt.Sprintf("%d", start.Unix())).
		EndTime(fmt.Sprintf("%d", end.Unix())).
		SortType("ByCreateTimeAsc").
		PageSize(50).
		Build()
	resp, err := c.sdk.Im.V1.Message.List(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("拉取群消息失败: %w", err)
	}
	if !resp.Success() {
		return nil, fmt.Errorf("拉取群消息失败: code=%d msg=%s", resp.Code, resp.Msg)
	}
	if resp.Data == nil {
		return nil, nil
	}

	messages := make([]Message, 0, len(resp.Data.Items))
	for _, item := range resp.Data.Items {
		if item == nil {
			continue
		}
		content, sender := "", ""
		if item.Body != nil {
			content = deref(item.Body.Content)
		}
		if item.Sender != nil {
			sender = deref(item.Sender.Id)
		}
		messages = append(messages, Message{
			MessageID: deref(item.MessageId),
			ChatID:    deref(item.ChatId),
			Sender:    sender,
			MsgType:   deref(item.MsgType),
			Content:   content,
			CreateAt:  deref(item.CreateTime),
		})
	}
	// TODO(分页): HasMore 为 true 时用 PageToken 继续拉，结案归档需要全量
	return messages, nil
}

// DownloadResource 下载消息里的图片 / 文件（多模态附件入对象存储用）。
func (c *Client) DownloadResource(ctx context.Context, messageID, fileKey, resType string) ([]byte, error) {
	if resType == "" {
		resType = "file"
	}
	req := larkim.NewGetMessageResourceReqBuilder().
		MessageId(messageID).
		FileKey(fileKey).
		Type(resType).
		Build()
	resp, err := c.sdk.Im.V1.MessageResource.Get(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("下载消息资源失败: %w", err)
	}
	if !resp.Success() {
		return nil, fmt.Errorf("下载消息资源失败: code=%d msg=%s", resp.Code, resp.Msg)
	}
	if resp.File == nil {
		return nil, nil
	}
	return io.ReadAll(resp.File)
}

// deref 取值（SDK 的可选字段都是指针）。
func deref(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
