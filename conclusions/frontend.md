# conclusions/frontend.md · 前端 / 静态演示页

## 技术评审多页演示的拆分（5 页 HTML 评审稿）

一句话：
按「总链路 → 分段」切 5 页自包含 HTML：01 总览与架构 / 02 知识库导入与向量化 / 03 预检RAG与L0建单 / 04 升级与群聊命令 / 05 结案回写与管理后台。每页独立上传分发，白底干净、图表放大。

实践：
- 内容拆分主线 = 评审讲解顺序（知识导入 → RAG 预检 → L0→L1/L2 升级 → 命令 → 总结回写），闭环能串起来又不重复。
- 数据表讲解按「每模块先讲有哪些表 / 字段关联 / 中间件与版本 / 为什么用它」。

注意：
- 分享内容只放 oncall 中台相关内容，历史残留（skills / requirement_*）与元叙述（引导句式、约定、交叉引用）全部剔除，评审现场不需要教学腔。
- 5 页各自独立：样式、Mermaid 库全部内联，不依赖目录共享资源，方便脱离目录打开 / 上传 / 转 PDF。

最佳实践：
- 源模板（干净可读）放 `review/_src`，共享样式 `review/css/style.css` 与库 `review/js/mermaid.min.js` 由 `review/_tools/build.py` 内联注入，产出项目根同名自包含 HTML —— 想改一页只改模板再重跑 build。
- 独立单页体积约 3.3 MB（大头是内联 mermaid.min.js）；想瘦身可改 build 输出为 CDN 引用。

限制：
- 内联库前需确认 `mermaid.min.js` 不含 `</script>`（含则要转义，否则截断 script）。
- `erDiagram` 关系语法 `||--o{` 里的 `o{` 不成对括号，静态括号校验会误报，别据此判错。

---

## 自包含 HTML 的构建（Python 占位注入）

一句话：
模板留 `<!--__CORE__-->` 占位，build.py 读样式 + mermaid 库，注入 style 与 script 成单文件，脚本里 `document.querySelectorAll('.mermaid')` 逐个 `mermaid.run()` 并在失败时把报错写回节点 —— 便于评审现场一眼看出哪张图坏了。

实践：
```python
html.replace("<!--__CORE__-->", "<style>"+css+"</style><script>"+lib+"</script>"+INIT_JS)
```
- mermaid init：`startOnLoad:false` + 手动 `mermaid.run({nodes:[el]})` + catch 显示错误文案。
- 放大可读：`themeVariables.fontSize:'18px'`、flowchart `nodeSpacing/rankSpacing` 调大、`useMaxWidth:false` + 外层容器 `overflow-x:auto`；打印 `@media print { .card,.mermaid-wrap{break-inside:avoid} }`。

注意：
- 中文路径 PowerShell 显示乱码属控制台编码问题，文件本身是 UTF-8 正常 —— 用 Python 读写时显式 `encoding='utf-8'`，不要依赖控制台输出判断文件名。

最佳实践：
- 一次改动模板 → 重建产物，改动点只存在模板一处；样式/脚本/内容分层占位，可整体替换。

限制：
- 内联 mermaid 使单文件大（3 MB+），不适合放 Git 频繁 diff；产物属生成物，改动应落在模板。
