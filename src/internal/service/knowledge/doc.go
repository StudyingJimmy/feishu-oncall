// Package knowledge 知识库服务：文档导入（原文进对象存储、切片进 ES、元数据进 MySQL）与检索。
//
// 待实现：
//   - Import：切分文档 -> 原文存对象存储 -> 切片批量写 ES -> 元数据落库（同一来源重复导入要覆盖）
//   - Search：预检用的检索入口；接向量检索时只改这里和 infra/es，上层不受影响
//   - ImportFromTicket：结案回写，把工单结论沉淀成文档，形成闭环
//   - 切分策略：按段落聚合 + 定长切分 + 重叠，避免上下文被截断
//
// 依赖方向：import repository、infra/es、infra/oss、model、conf、pkg。
package knowledge
