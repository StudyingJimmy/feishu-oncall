// Package precheck 预检服务：飞书卡片引导用户自查，检索知识库给建议，没解决则转人工。
//
// 待实现：
//   - Start：建预检会话 + 发第一张问题卡片
//   - 答题推进：记录答案、推进步骤、原地更新卡片
//   - 出建议：用答案拼检索词调知识库，命中则给自助方案，未命中直接引导转人工
//   - 已解决 / 没解决分支：前者标记自助解决；后者定优先级、选团队、交给 dispatch 建单拉群
//   - 消息入口：私聊或群里 @机器人 触发；图片、文件落对象存储作为附件线索
//
// 依赖方向：import repository、model、conf，以及同层 service（knowledge / dispatch / notify）与 infra。
package precheck
