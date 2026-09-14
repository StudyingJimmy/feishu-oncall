// Package dispatch 派单服务：选值班团队 -> 拉服务群 -> 拉值班同学 -> 发工单卡片。
//
// 待实现：
//   - ChooseTeam：按问题类型 / 关键词匹配团队（规则放 conf），匹配不到用兜底团队
//   - CreateServiceGroup：建群，拉提单人与当前值班同学（排班取不到就退化到团队成员）
//   - Dispatch：建单 -> 拉群 -> 回写 chat_id -> 群内发工单卡片
//   - 拉群失败要保留工单（不丢单），并私聊提醒值班同学
//
// 依赖方向：import repository、model、conf、service/ticket、service/notify、infra/lark。
package dispatch
