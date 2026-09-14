// Package ticket 工单服务：状态机 + 全过程时间线 + 事件发布。
//
// 待实现：
//   - Create / Claim / Escalate / Resolve / Close / Reopen / AddNote
//   - 统一状态流转入口：校验流转白名单、维护时间戳、写时间线、发事件（避免状态改了但时间线没记）
//   - RecordMessage：把服务群里的消息记进全过程时间线
//   - HandleCardAction：处理工单卡片按钮（认领 / 升级 / 已解决 / 结案 / 重开）
//   - Close 时回写知识库，把结论沉淀成文档，下次预检就能命中
//
// 依赖方向：import repository、model、conf、service/notify、service/knowledge、infra/mq。
package ticket
