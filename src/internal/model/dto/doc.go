// Package dto 层间传输对象：入站事件、卡片 value、各服务入参出参。
//
// 与 entity 的区别：entity 落库，dto 只在内存里传递。
//
// 已实现：
//   - event.go   入站事件归一化：Event / MessageEvent / CardActionEvent / CardActionValue
//
// 待实现（写业务时补）：
//   - precheck.go 预检发起入参、知识库命中、预检结果
//   - ticket.go   建单入参、工单卡片视图、时间线展示项、列表过滤条件
//
// 依赖方向：可以引用 enum，不 import repository / service / infra。
package dto
