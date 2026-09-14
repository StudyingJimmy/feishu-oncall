// 预检卡片模板：卡片 JSON 只在这里拼，service 负责挑模板、填数据。
//
// 用卡片 JSON 1.0 结构（config / header / elements）；按钮的 value 就是回调时带回的 dto.CardActionValue。
package precheck

import (
	"fmt"
	"strings"

	"bokeoncall/internal/model/dto"
)

// atUser @某人：机器人每次回复的第一句都要 @ 发起人。
func atUser(openID string) string {
	if openID == "" {
		return ""
	}
	return fmt.Sprintf("<at id=%s></at> ", openID)
}

// baseCard 卡片骨架：宽屏 + 支持原地更新。
func baseCard(title, template string, elements ...map[string]any) map[string]any {
	return map[string]any{
		"config": map[string]any{"wide_screen_mode": true, "update_multi": true},
		"header": map[string]any{
			"template": template,
			"title":    map[string]any{"tag": "plain_text", "content": title},
		},
		"elements": elements,
	}
}

func mdBlock(content string) map[string]any {
	return map[string]any{"tag": "div", "text": map[string]any{"tag": "lark_md", "content": content}}
}

func divider() map[string]any { return map[string]any{"tag": "hr"} }

func actionBlock(buttons ...map[string]any) map[string]any {
	return map[string]any{"tag": "action", "actions": buttons}
}

// button 卡片按钮。btnType：default / primary / danger。
func button(text, btnType string, value dto.CardActionValue) map[string]any {
	if btnType == "" {
		btnType = "default"
	}
	return map[string]any{
		"tag":   "button",
		"text":  map[string]any{"tag": "plain_text", "content": text},
		"type":  btnType,
		"value": actionValue(value),
	}
}

// SearchingCard 检索中的卡片：先 @发起人，提示正在匹配知识库；分割线下给「跳过，转人工」。
func SearchingCard(requesterOpenID, sessionID string, ephemeral bool) map[string]any {
	return baseCard("BokeOnCall 预检助手", "blue",
		mdBlock(atUser(requesterOpenID)+"正在匹配知识库..."),
		divider(),
		actionBlock(button("跳过，转人工", "danger", dto.CardActionValue{
			Action:    dto.ActionPrecheckSkip,
			SessionID: sessionID,
			Ephemeral: ephemeral,
		})),
	)
}

// ResultCard 结果卡片：结果下方给「👍 有帮助 / 👎 无帮助」，分割线下给「已解决 / 仍需转人工」。
func ResultCard(requesterOpenID, sessionID string, hits []dto.KbHit, ephemeral bool) map[string]any {
	return baseCard("BokeOnCall 预检结果", "green",
		mdBlock(atUser(requesterOpenID)+resultText(hits)),
		actionBlock(
			button("👍 有帮助", "primary", dto.CardActionValue{Action: dto.ActionPrecheckHelpful, SessionID: sessionID, Ephemeral: ephemeral}),
			button("👎 无帮助", "default", dto.CardActionValue{Action: dto.ActionPrecheckUseless, SessionID: sessionID, Ephemeral: ephemeral}),
		),
		divider(),
		actionBlock(
			button("已解决，无需转人工", "primary", dto.CardActionValue{Action: dto.ActionPrecheckSolved, SessionID: sessionID, Ephemeral: ephemeral}),
			button("仍需转人工", "danger", dto.CardActionValue{Action: dto.ActionPrecheckEscalate, SessionID: sessionID, Ephemeral: ephemeral}),
		),
	)
}

// FeedbackCard 反馈已记录：用户点了「有帮助 / 无帮助」后替换成这张卡片。
func FeedbackCard(requesterOpenID, sessionID, choice string, ephemeral bool) map[string]any {
	return baseCard("感谢反馈", "grey",
		mdBlock(atUser(requesterOpenID)+fmt.Sprintf("您已选择 %s，感谢您的反馈！", choice)),
		divider(),
		actionBlock(
			button("已解决，无需转人工", "primary", dto.CardActionValue{Action: dto.ActionPrecheckSolved, SessionID: sessionID, Ephemeral: ephemeral}),
			button("仍需转人工", "danger", dto.CardActionValue{Action: dto.ActionPrecheckEscalate, SessionID: sessionID, Ephemeral: ephemeral}),
		),
	)
}

// BuildGroupCard 建群卡片：转人工后让用户补问题描述与目标租户，提交后去建群。
func BuildGroupCard(requesterOpenID, sessionID string, ephemeral bool) map[string]any {
	submit := map[string]any{
		"tag":         "button",
		"action_type": "form_submit", // 与表单容器绑定，点击后提交整张表单
		"name":        "submit",
		"text":        map[string]any{"tag": "lark_md", "content": "提交并建群"},
		"type":        "primary",
		"value": actionValue(dto.CardActionValue{
			Action:    dto.ActionPrecheckBuildGroup,
			SessionID: sessionID,
			Ephemeral: ephemeral,
		}),
	}
	form := map[string]any{
		"tag":  "form",
		"name": "build_group_form",
		"elements": []map[string]any{
			{
				"tag":         "input",
				"name":        "description", // 表单字段名，回调时在 form_value 里带回来
				"label":       map[string]any{"tag": "plain_text", "content": "问题描述"},
				"placeholder": map[string]any{"tag": "plain_text", "content": "简单描述现象与影响范围"},
				"input_type":  "multiline_text",
				"rows":        3,
				"max_length":  500,
				"required":    true,
			},
			{
				"tag":         "select_static",
				"name":        "tenant",
				"placeholder": map[string]any{"tag": "plain_text", "content": "选择目标租户"},
				"options":     tenantOptions(),
			},
			submit,
		},
	}
	return baseCard("建群信息", "blue",
		mdBlock(atUser(requesterOpenID)+"请补充问题描述和目标租户，我会拉值班同学进群。"),
		form,
	)
}

// tenantOptions 目标租户下拉选项。
//
// TODO(数据源): 现在写死几个占位选项，后续从租户服务 / 配置拉取真实列表。
// 注意：飞书卡片 JSON 的下拉选择没有"可搜索"字段（1.0 / 2.0 组件文档都没有），
// 选项多时建议改成「输入关键词 + 提交后校验」，或改用人员选择器 select_person（自带搜索）。
func tenantOptions() []map[string]any {
	tenants := []struct{ Text, Value string }{
		{"示例租户 A", "tenant_a"},
		{"示例租户 B", "tenant_b"},
		{"示例租户 C", "tenant_c"},
	}
	options := make([]map[string]any, 0, len(tenants))
	for _, tenant := range tenants {
		options = append(options, map[string]any{
			"text":  map[string]any{"tag": "plain_text", "content": tenant.Text},
			"value": tenant.Value,
		})
	}
	return options
}

// resultText 命中知识库时列出结果；没命中直接引导转人工。
func resultText(hits []dto.KbHit) string {
	if len(hits) == 0 {
		return "未检索到相关结果，请尝试转人工。"
	}
	var sb strings.Builder
	sb.WriteString("**可能相关的处理方案**\n")
	for _, hit := range hits {
		fmt.Fprintf(&sb, "- %s\n  %s\n", hit.Title, snippet(hit.Snippet))
	}
	return sb.String()
}

func snippet(text string) string {
	text = strings.TrimSpace(strings.ReplaceAll(text, "\n", " "))
	runes := []rune(text)
	if len(runes) <= 120 {
		return text
	}
	return string(runes[:120]) + "..."
}

// actionValue 把动作载荷转成按钮 value（回调时原样带回）。
func actionValue(v dto.CardActionValue) map[string]any {
	value := map[string]any{"action": v.Action}
	if v.SessionID != "" {
		value["session_id"] = v.SessionID
	}
	if v.TicketNo != "" {
		value["ticket_no"] = v.TicketNo
	}
	if v.TeamKey != "" {
		value["team_key"] = v.TeamKey
	}
	if v.Answer != "" {
		value["answer"] = v.Answer
	}
	if v.Ephemeral {
		value["ephemeral"] = true
	}
	return value
}
