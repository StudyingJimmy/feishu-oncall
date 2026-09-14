// Package entity 数据库实体（表结构），建议用 GORM 模型。
//
// 待实现（表名统一 oc_ 前缀）：
//   - oc_ticket           工单主表：工单号、标题、状态、优先级、提单人、服务群 chat_id、负责人、SLA 截止时间
//   - oc_ticket_event     全过程时间线：每次状态变化 / 群消息 / 备注各一条
//   - oc_precheck_session 预检会话：问答进度、答案、检索命中、关联工单号
//   - oc_team / oc_team_member / oc_duty_shift  值班团队、成员、排班
//   - oc_kb_document / oc_kb_chunk               知识库文档元数据与切片元数据
//   - oc_processed_event  事件幂等表（飞书会重推事件）
//
// 约定：只放表结构与字段，不放业务方法；时间统一存 UTC，展示层再转东八区。
// 依赖方向：可以引用 enum，不 import repository / service / infra。
package entity
