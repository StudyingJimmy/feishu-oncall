// Package handler HTTP 处理器。
//
// 待实现：
//   - lark_event：URL 校验（challenge）、事件解密、token 校验、事件幂等（同一 event_id 只处理一次），
//     再按事件类型分发：消息事件（命令 / 预检 / 全过程归档）、卡片回调（按 action 分发到预检或工单）
//   - health：/healthz 存活探针，/readyz 依赖探活（MySQL、ES、对象存储）
//
// 依赖方向：import service、infra、model/dto、conf。
package handler
