// Package tenant 租户数据源（占位实现）。
//
// TODO: 接真实租户服务 / 配置；现在返回几个写死的选项，供"目标租户"下拉选择使用。
package tenant

// Item 租户。
type Item struct {
	ID   string
	Name string
}

// List 租户列表。
func List() []Item {
	return []Item{
		{ID: "tenant_a", Name: "示例租户 A"},
		{ID: "tenant_b", Name: "示例租户 B"},
		{ID: "tenant_c", Name: "示例租户 C"},
	}
}

// SelectOptions 转成飞书下拉选择（select_static）需要的 options。
func SelectOptions() []map[string]any {
	items := List()
	options := make([]map[string]any, 0, len(items))
	for _, item := range items {
		options = append(options, map[string]any{
			"text":  map[string]any{"tag": "plain_text", "content": item.Name},
			"value": item.ID,
		})
	}
	return options
}
