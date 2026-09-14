// Package view 工单视图组装：把工单、团队、时间线拼成卡片与接口需要的视图对象。
//
// 单独成包的原因：ticket 与 notify 都要用这个视图，抽出来可避免它们互相依赖。
//
// 待实现：Build(ticketNo, timelineLimit) -> dto.TicketView（含负责人、团队名、最近 N 条进展）。
// 依赖方向：import repository、model/dto。
package view
