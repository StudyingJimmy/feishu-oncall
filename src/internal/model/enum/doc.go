// Package enum 领域枚举：状态、优先级、事件类型、卡片动作、角色与层级。
//
// 待实现：
//   - TicketStatus：precheck / pending_dispatch / pending_claim / in_progress / escalated / resolved / closed / cancelled
//   - Priority：P0 ~ P3（线上不可用 -> 仅个人受影响）
//   - EventType：全过程时间线的事件类型（建单、拉群、认领、升级、群消息、结案等）
//   - CardAction：卡片按钮动作（预检提交、认领、升级、解决、结案）
//   - PrecheckStatus / TeamRole / DutyLevel / Source
//
// 注意：这些值会落库、写进卡片 value、进消息队列，属于对外契约，改动要谨慎。
// 依赖方向：只依赖标准库。
package enum
