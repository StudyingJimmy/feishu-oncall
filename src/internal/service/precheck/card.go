// 预检卡片模板：卡片 JSON 只在这里拼，service 负责挑模板、填数据。
//
// 用卡片 JSON 1.0 结构（config / header / elements）；按钮的 value 就是回调时带回的 dto.CardActionValue。
// 交互后的更新通过"卡片回调响应体里回传新卡片"完成（见 infra/lark/dispatcher.go）。
package precheck

import (
	"fmt"
	"strings"

	"bokeoncall/internal/model/dto"
	"bokeoncall/internal/service/tenant"
)

// atUser @某人：机器人每次回复的第一句都要 @ 发起人。
func atUser(openID string) string {
	if openID == "" {
		return ""
	}
	return fmt.Sprintf("<at id=%s></at> ", openID)
}

// baseCard 卡片骨架：宽屏 + 支持更新。
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
func SearchingCard(requesterOpenID, sessionID string) map[string]any {
	return baseCard("BokeOnCall 预检助手", "blue",
		mdBlock(atUser(requesterOpenID)+"正在匹配知识库..."),
		divider(),
		actionBlock(button("跳过，转人工", "danger", dto.CardActionValue{
			Action:    dto.ActionPrecheckSkip,
			SessionID: sessionID,
		})),
	)
}

// ResultCard 结果卡片，一行一个状态：
//
//	@发起人 <结果内容>
//	[👍 有帮助] [👎 无帮助]            <- feedback 非空时，这一行变成一句反馈文案
//	────────────
//	[已解决，无需转人工] [仍需转人工]   <- decision 非空时，这一行变成一句决策文案
//
// 为了不在服务端存状态，按钮 value 里带上 result / feedback / decision，点击时原样带回，
// 据此重建整张卡片（见 dto.CardActionValue）。
func ResultCard(requesterOpenID, sessionID, result, feedback, decision string) map[string]any {
	elements := []map[string]any{mdBlock(atUser(requesterOpenID) + result)}

	if feedback != "" {
		elements = append(elements, mdBlock(feedback))
	} else {
		value := dto.CardActionValue{SessionID: sessionID, Result: result, Feedback: feedback, Decision: decision}
		helpful, useless := value, value
		helpful.Action = dto.ActionPrecheckHelpful
		useless.Action = dto.ActionPrecheckUseless
		elements = append(elements, actionBlock(
			button("👍 有帮助", "default", helpful),
			button("👎 无帮助", "default", useless),
		))
	}

	elements = append(elements, divider())
	if decision != "" {
		elements = append(elements, mdBlock(decision))
	} else {
		value := dto.CardActionValue{SessionID: sessionID, Result: result, Feedback: feedback, Decision: decision}
		solved, escalate := value, value
		solved.Action = dto.ActionPrecheckSolved
		escalate.Action = dto.ActionPrecheckEscalate
		elements = append(elements, actionBlock(
			button("已解决，无需转人工", "primary", solved),
			button("仍需转人工", "danger", escalate),
		))
	}
	return baseCard("BokeOnCall 预检结果", "green", elements...)
}

// BuildGroupCard 建群卡片：转人工后让用户补问题描述与目标租户，提交后去建群。
func BuildGroupCard(requesterOpenID, sessionID string) map[string]any {
	submit := map[string]any{
		"tag":         "button",
		"action_type": "form_submit", // 与表单容器绑定，点击后提交整张表单
		"name":        "submit",
		"text":        map[string]any{"tag": "lark_md", "content": "确认提交"},
		"type":        "primary",
		"value":       actionValue(dto.CardActionValue{Action: dto.ActionPrecheckBuildGroup, SessionID: sessionID}),
	}
	cancel := map[string]any{
		"tag":   "button",
		"name":  "cancel", // 表单容器内的组件都要有 name
		"text":  map[string]any{"tag": "lark_md", "content": "取消"},
		"type":  "danger",
		"value": actionValue(dto.CardActionValue{Action: dto.ActionPrecheckCancel, SessionID: sessionID}),
	}
	// 用分栏把「确认提交」和「取消」放在同一行（表单内的按钮需要裹在分栏里才能并排）。
	// flex_mode=none + 列宽 auto：列宽跟随按钮本身，两个按钮挨在一起。
	buttonRow := map[string]any{
		"tag":                "column_set",
		"flex_mode":          "none",
		"horizontal_spacing": "small", // 4px
		"background_style":   "default",
		"columns": []map[string]any{
			{"tag": "column", "width": "auto", "vertical_align": "top", "elements": []map[string]any{submit}},
			{"tag": "column", "width": "auto", "vertical_align": "top", "elements": []map[string]any{cancel}},
		},
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
				"options":     tenant.SelectOptions(),
			},
			buttonRow,
		},
	}
	return baseCard("请确认建群信息", "blue",
		mdBlock(atUser(requesterOpenID)+"请补充问题描述和目标租户，我会拉值班同学进群。"),
		form,
	)
}

// BuildGroupPendingCard 建群过程中的卡片：只显示一行状态（已提交 / 正在建群 / 建群失败）。
func BuildGroupPendingCard(requesterOpenID, text string) map[string]any {
	return baseCard("建群信息", "blue", mdBlock(atUser(requesterOpenID)+text))
}

// BuildGroupDoneCard 建群成功的卡片：展示提交内容 + 「前往群聊」按钮。
func BuildGroupDoneCard(requesterOpenID, chatID, description, tenant string) map[string]any {
	content := fmt.Sprintf("**已提交，建群成功**\n\n问题描述：%s\n目标租户：%s", description, tenant)
	elements := []map[string]any{mdBlock(atUser(requesterOpenID) + content)}
	if chatID != "" {
		elements = append(elements, divider(), actionBlock(groupLinkButton(chatID)))
	}
	return baseCard("建群信息", "green", elements...)
}

// groupLinkButton 「前往群聊」按钮：用飞书 applink 直达会话。
func groupLinkButton(chatID string) map[string]any {
	return map[string]any{
		"tag":  "button",
		"text": map[string]any{"tag": "plain_text", "content": "前往群聊"},
		"type": "primary",
		"url":  fmt.Sprintf("https://applink.feishu.cn/client/chat/open?openChatId=%s", chatID),
	}
}

// WorkTimeCard 非工作时间提示卡（发在群聊里，公开可见）。
// 点「需要紧急介入」时会带上租户与问题描述，回调时据此拉值班人员入群。
func WorkTimeCard(workTimeText, tenantID, description string) map[string]any {
	content := fmt.Sprintf(
		"当前工作时间为%s。\n您可以先在群聊中更加详细地描述问题现状，待至工作时间值班人员会自动入群进行处理。\n如果情况紧急，您也可以点击下方按钮立即请求值班人员介入处理。",
		workTimeText)
	return baseCard("非工作时间提示", "orange",
		mdBlock(content),
		divider(),
		actionBlock(button("需要紧急介入", "danger", dto.CardActionValue{
			Action:      dto.ActionPrecheckUrgent,
			TenantID:    tenantID,
			Description: description,
		})),
	)
}

// UrgentRequestedCard 已收到紧急介入请求（替换掉按钮，避免重复点击）。
func UrgentRequestedCard(workTimeText string) map[string]any {
	return baseCard("非工作时间提示", "orange", mdBlock(fmt.Sprintf(
		"当前工作时间为%s。\n已收到紧急介入请求，正在通知值班人员入群，请稍候。", workTimeText)))
}

// WorkingCard 工单处理中（发在群聊里，公开可见）：@值班人员 + 问题描述 + 本群可用命令。
func WorkingCard(onDutyOpenIDs []string, description string) map[string]any {
	mention := "@值班同学（待接入值班表）"
	if len(onDutyOpenIDs) > 0 {
		parts := make([]string, 0, len(onDutyOpenIDs))
		for _, openID := range onDutyOpenIDs {
			parts = append(parts, fmt.Sprintf("<at id=%s></at>", openID))
		}
		mention = strings.Join(parts, " ")
	}
	content := fmt.Sprintf("%s\n问题描述：%s\n\n**本群可用命令**\n/upgrade 升级到上一层值班\n/transfer 转派给其它团队或同学\n/event 升级为故障、拉起作战室\n/solve 标记已解决\n/save 暂时挂起",
		mention, description)
	return baseCard("工单处理中", "blue", mdBlock(content))
}

// shorten 截断过长文本（群名等场景用）。
func shorten(text string, max int) string {
	text = strings.TrimSpace(text)
	runes := []rune(text)
	if len(runes) <= max {
		return text
	}
	return string(runes[:max])
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
	if v.Result != "" {
		value["result"] = v.Result
	}
	if v.Feedback != "" {
		value["feedback"] = v.Feedback
	}
	if v.Decision != "" {
		value["decision"] = v.Decision
	}
	if v.TenantID != "" {
		value["tenant_id"] = v.TenantID
	}
	if v.Description != "" {
		value["description"] = v.Description
	}
	return value
}
