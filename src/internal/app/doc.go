// Package app 依赖装配：把 conf、infra、repository、service 组装成一个容器供 cmd 使用。
//
// 待实现：
//   - Build：连 MySQL、ES、对象存储、飞书、消息队列，构造各 repository 与 service
//   - 启动时初始化索引、桶、默认团队数据（幂等，失败只告警不阻断启动）
//   - Close：释放连接
//
// 这样 cmd/server 与 cmd/worker 只需调用 Build，不会重复初始化逻辑。
package app
