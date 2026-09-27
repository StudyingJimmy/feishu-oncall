# Boke Oncall 前端

React + TypeScript + Ant Design + Vite 实现的飞书应用网页界面。`index.html` 只是应用入口，页面由组件构建。

## 本地运行

```powershell
cd app
npm.cmd install
npm.cmd run dev
```

访问 `http://127.0.0.1:5173`。在 PowerShell 禁止执行 `npm.ps1` 的环境下使用 `npm.cmd`；其他终端可直接用 `npm`。

```powershell
npm.cmd run typecheck
npm.cmd run lint
npm.cmd run build
npm.cmd run preview
```

构建产物位于 `app/dist/`，可部署到支持 SPA 静态资源的服务。该项目目前没有飞书应用身份认证或真实 API，不能把 `dist` 当作已完成的生产应用。

## 结构

| 路径 | 职责 |
| --- | --- |
| `src/workspace/Workspace.tsx` | 页面状态与交互编排 |
| `src/components/` | 侧栏、对话、输入框、租户排行、工单详情、确认弹窗 |
| `src/domain/types.ts` | 领域类型与 `OncallGateway` 契约 |
| `src/services/mockOncallGateway.ts` | 模拟租户、知识库、工单及建单逻辑 |
| `src/styles.css` | 工作台视觉与响应式样式 |

## 演示交互

1. 输入问题时，租户排行会按关键词实时更新。
2. 按 Enter 或点击“搜索”会把问题加入对话，并返回模拟知识库结果和推荐租户；Shift + Enter 换行。
3. 点击租户，填写问题描述并确认，模拟拉群建单。新工单会出现在“进行中”侧栏并保存在当前浏览器 `localStorage`。
4. 侧栏按“进行中 / 已完成”展开收起，点击单号查看工单详情。

真实接入方案与可复制的后续实现提示词见仓库根目录 [readme.md](../readme.md)。
