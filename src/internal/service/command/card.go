// 命令卡片模板：/upgrade /transfer /event /save 各一张表单卡片（仅发起人可见）。
//
// 卡片结构：标题 + 表单（若干输入项 + 同一行的「确认提交 / 取消」）。
// 这里自带一份卡片骨架工具函数（与 precheck 包各自独立，避免为几个小函数再抽一层）。
package command

import (
	"bokeoncall/internal/model/dto"
	"bokeoncall/internal/service/tenant"
)

// UpgradeCard 升级：只需填写具体原因。
func UpgradeCard() map[string]any {
	return formCard("请填写本次升级的信息", dto.ActionCommandUpgrade,
		inputField("reason", "具体原因", "为什么要升级，例如：影响面扩大 / L1 无法定位"),
	)
}

// TransferCard 转接：先填目标租户，再填具体原因。
func TransferCard() map[string]any {
	return formCard("请填写本次转接的信息", dto.ActionCommandTransfer,
		tenantField(),
		inputField("reason", "具体原因", "为什么要转接给该租户"),
	)
}

// EventCard 升级作战室：作战室群名 + 关联项目 + 具体原因。
func EventCard() map[string]any {
	return formCard("请填写本次升级作战室的信息", dto.ActionCommandEvent,
		inputField("room_name", "作战室群名", "例如：XX 故障作战室"),
		inputField("project", "关联项目", "例如：知识库检索"),
		inputField("reason", "具体原因", "故障现象与影响范围"),
	)
}

// SaveCard 工单挂起：只需填写具体原因。
func SaveCard() map[string]any {
	return formCard("请填写本次工单挂起的信息", dto.ActionCommandSave,
		inputField("reason", "具体原因", "为什么要挂起，例如：等上游修复"),
	)
}

// formCard 卡片骨架：标题 + 表单（字段 + 确认提交/取消）。
func formCard(title, submitAction string, fields ...map[string]any) map[string]any {
	submit := map[string]any{
		"tag":         "button",
		"action_type": "form_submit", // 与表单容器绑定
		"name":        "submit",
		"text":        map[string]any{"tag": "lark_md", "content": "确认提交"},
		"type":        "primary",
		"value":       map[string]any{"action": submitAction},
	}
	cancel := map[string]any{
		"tag":   "button",
		"name":  "cancel",
		"text":  map[string]any{"tag": "lark_md", "content": "取消"},
		"type":  "danger",
		"value": map[string]any{"action": dto.ActionCommandCancel},
	}
	form := map[string]any{
		"tag":      "form",
		"name":     "command_form",
		"elements": append(fields, buttonRow(submit, cancel)),
	}
	return map[string]any{
		"config": map[string]any{"wide_screen_mode": true, "update_multi": true},
		"header": map[string]any{
			"template": "blue",
			"title":    map[string]any{"tag": "plain_text", "content": title},
		},
		"elements": []map[string]any{form},
	}
}

// buttonRow 用分栏把两个按钮放在同一行。
// flex_mode=none + 列宽 auto：列宽跟随按钮本身，两个按钮挨在一起（用 bisect 会各占半行、间距很大）。
func buttonRow(buttons ...map[string]any) map[string]any {
	columns := make([]map[string]any, 0, len(buttons))
	for _, item := range buttons {
		columns = append(columns, map[string]any{
			"tag": "column", "width": "auto", "vertical_align": "top",
			"elements": []map[string]any{item},
		})
	}
	return map[string]any{
		"tag":                "column_set",
		"flex_mode":          "none",
		"horizontal_spacing": "small", // 4px
		"background_style":   "default",
		"columns":            columns,
	}
}

// inputField 多行输入框（表单字段，name 会出现在回调的 form_value 里）。
func inputField(name, label, placeholder string) map[string]any {
	return map[string]any{
		"tag":         "input",
		"name":        name,
		"label":       map[string]any{"tag": "plain_text", "content": label},
		"placeholder": map[string]any{"tag": "plain_text", "content": placeholder},
		"input_type":  "multiline_text",
		"rows":        2,
		"max_length":  500,
		"required":    true,
	}
}

// tenantField 目标租户下拉选择。
func tenantField() map[string]any {
	return map[string]any{
		"tag":         "select_static",
		"name":        "tenant",
		"placeholder": map[string]any{"tag": "plain_text", "content": "选择目标租户"},
		"options":     tenant.SelectOptions(),
	}
}
