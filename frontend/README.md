# NovelOS 前端文件说明

这里是 NovelOS MVP 的 React + TypeScript 前端。当前前端按 OpenAPI 资源域组织代码，并围绕最小创作闭环推进：项目、作者圣经、角色、关系、story run、commit 和章节。

## 根目录文件

- `package.json`：前端包信息、依赖和脚本。Windows PowerShell 拦截 `pnpm.ps1` 时，用 `pnpm.cmd run build`。
- `pnpm-lock.yaml`：pnpm 生成的精确依赖锁文件。
- `index.html`：Vite HTML 入口，提供 `src/main.tsx` 挂载用的 `#root` 节点。
- `vite.config.ts`：Vite 和 React 插件配置；开发时把本地 `/api` 请求代理到 `http://localhost:8000`。
- `tsconfig.json`：浏览器端源码的 TypeScript 配置，覆盖 `src/`。
- `tsconfig.node.json`：Node 侧工具文件的 TypeScript 配置，例如 `vite.config.ts`。
- `dist/`：生产构建输出目录，不是源码。
- `node_modules/`：已安装依赖目录，不是源码。

## 源码结构

- `src/main.tsx`：React 入口，安装 Router 和 TanStack Query。
- `src/app/router.tsx`：路由表，包含首页、项目工作台、详情页和 404。
- `src/app/queryClient.ts`：共享的 TanStack Query 默认配置。
- `src/api/`：每个 OpenAPI 资源组一个 API client，另有共享的 `http.ts`。
- `src/types/api.ts`：前端 API 类型，尽量对齐当前 OpenAPI 契约。
- `src/hooks/`：SSE 和最近项目 localStorage 的复用 hook。
- `src/components/layout/`：应用外壳、项目外壳和项目导航。
- `src/components/feedback/`：加载、错误和空状态组件。
- `src/components/forms/`：数组字段、世界状态、角色选择等复用表单控件。
- `src/features/`：按业务域分组的路由级页面。
- `src/styles/global.css`：布局、表单、工作台和响应式规则的全局样式。

## 页面文件

- `features/projects/HomePage.tsx`：创建/打开项目，以及最近项目快捷入口。
- `features/projects/ProjectOverviewPage.tsx`：项目统计、项目编辑和工作区入口。
- `features/authorBible/AuthorBiblePage.tsx`：作者圣经读取、编辑和保存。
- `features/characters/CharactersPage.tsx`：角色列表和创建表单。
- `features/characters/CharacterDetailPage.tsx`：角色详情编辑和角色记忆创建。
- `features/relationships/RelationshipsPage.tsx`：关系列表和创建表单。
- `features/setup/SetupWorkspacePage.tsx`：setup session/run 流程和应用正式设定动作。
- `features/story/StoryWorkspacePage.tsx`：story session/run 流程、SSE 正文展示和 commit 动作。
- `features/story/useStoryRunEvents.ts`：story run 专用 SSE 状态整理逻辑。
- `features/chapters/ChaptersPage.tsx`：已提交章节列表。
- `features/chapters/ChapterDetailPage.tsx`：已提交章节阅读页。
- `features/system/NotFoundPage.tsx`：未匹配路由的兜底页面。

## 后端连接

浏览器会请求前端相对路径，例如 `/api/v1/projects`。运行 `pnpm run dev` 时，Vite 会把 `/api` 代理到 `http://localhost:8000`。如果后端端口不同，需要修改 `vite.config.ts`。

## 最小流程目标

MVP 最小流程完成标准是：不手动改数据库，可以跑通下面这条链路。

```txt
创建项目
  -> 编辑作者圣经
  -> 创建角色和关系
  -> 创建 story session
  -> advance story
  -> 查看 story run result
  -> commit
  -> 阅读已提交章节
```
