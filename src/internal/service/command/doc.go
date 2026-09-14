// Package command 群内文本命令：让值班同学不点卡片也能操作工单。
//
// 已实现：
//   - parser.go：命令解析（/upgrade /transfer /event /solve /save），路由与处理共用
//   - handler.go：命令入口与 5 个处理函数（目前是日志占位）
//
// 命令语义：
//   - /upgrade  升级工单到上一层值班
//   - /transfer 转派给其它团队或同学
//   - /event    升级为故障，拉起故障作战室
//   - /solve    标记工单已解决
//   - /save     暂时挂起工单
//
// 待实现：
//   - 各处理函数接 ticket service 改工单状态、接 lark client 拉作战室/发通知
//   - 执行前校验操作人是否有权限（是否该工单群成员 / 值班同学）
//
// 依赖方向：import service/ticket、model、conf。router 会 import 本包做命令识别。
package command
