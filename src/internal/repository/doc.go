// Package repository 数据访问层：唯一直接操作数据库的一层。
//
// 待实现（每个实体一个 repo 文件）：
//   - ticket_repo    建单、按工单号 / 群 ID 查询、状态更新、时间线追加与查询、超时工单扫描
//   - precheck_repo  会话创建 / 查询 / 推进、结束、超时会话扫描
//   - team_repo      团队与成员查询、某时刻在值班的人、排班覆盖写入
//   - knowledge_repo 文档元数据 upsert、切片元数据覆盖写入与查询
//   - event_repo     事件幂等（用唯一键 + OnConflict DoNothing 抢占）
//
// 约定：方法第一参数都是 context；只做增删改查，不写业务判断（业务判断在 service）。
// 依赖方向：import model/entity、infra/mysql，不 import service / api。
package repository
