// Package task 后台任务：worker 进程里跑的定时与异步逻辑。
//
// 待实现：
//   - sla_watchdog 扫描超时工单：未认领的自动升级，已认领的催办（同一条只提醒一次）
//   - roster_sync  从飞书日历 / 多维表格同步值班表到 oc_duty_shift
//   - worker       消费工单全过程事件流（归档、报表、知识沉淀兜底补偿）
//
// 依赖方向：import service、repository、infra/mq、conf。
package task
