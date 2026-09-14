// Package notify 通知服务：把状态变化渲染成卡片发到群里 / 私聊。
//
// 待实现：
//   - 发卡片 / 回复卡片 / 原地更新卡片（预检步骤切换、工单进展）
//   - 工单卡片、结案卡片、预检问题卡片与建议卡片的渲染（卡片 JSON 建议单独放本包 card.go）
//   - 提醒值班同学、按 open_id 查姓名（失败降级，不阻塞主流程）
//
// 只负责渲染与发送，不做状态流转判断（那是 service/ticket 的事）。
// 依赖方向：import infra/lark、repository、model、conf。
package notify
