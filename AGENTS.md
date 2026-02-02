# Maxx (AI API Proxy Gateway) - Codex CLI 项目指引

本文件用于让 Codex CLI 快速理解本仓库的技术栈、目录结构与常用工作流；默认以中文进行沟通与输出。

## 项目概述

Maxx 是一个多提供商 AI 代理网关，支持 Claude、OpenAI、Gemini 和 Codex 协议。内置管理界面、路由策略、使用统计和计费功能。支持 Docker 部署和 Wails 桌面应用。

## 技术栈

### 后端 (Go)
- **语言**: Go 1.25+
- **框架**: 标准库 `net/http` + `http.ServeMux` (无大型 Web 框架)
- **ORM**: GORM (SQLite / MySQL / PostgreSQL)
- **桌面框架**: Wails v2
- **其他**: JWT 认证

### 前端 (React)
- **框架**: React 19 + TypeScript
- **构建**: Vite
- **样式**: TailwindCSS 4
- **UI 组件**: shadcn/ui + Radix UI (@base-ui)
- **状态管理**: TanStack Query (React Query) + Zustand
- **路由**: React Router v7

## 目录结构速览

- `cmd/`: 应用程序入口 (`maxx/main.go`)
- `internal/`: 核心业务代码
    - `adapter/`: 协议适配器 (Client/Provider)，负责请求/响应的格式转换与适配
    - `core/`: 核心服务 (Server, Database, Task)
    - `domain/`: 领域模型定义 (Model, Repository Interface)
    - `handler/`: HTTP 路由处理
    - `repository/`: 数据库访问层 (SQLite/MySQL/PostgreSQL 实现)
    - `service/`: 业务逻辑服务 (Admin, Backup, Task)
    - `desktop/`: 桌面应用特定逻辑 (Launcher, Tray)
- `web/`: 前端 React 项目
    - `src/components/`: UI 组件
    - `src/pages/`: 页面组件
    - `src/hooks/`: 自定义 Hooks (Query, Mutation)
    - `src/lib/`: 工具库 (Transport 层抽象了 HTTP/Wails 通信)
- `build/`: 构建产物和资源
- `.github/`: CI/CD 工作流

## 编码约定

- **语言**: 推荐在代码注释与文档使用中文；代码变量命名使用英文。
- **Go**: 遵循标准 Go 代码规范 (`gofmt`)。
    - 错误处理：显式处理错误，避免忽略错误。
    - 依赖注入：主要通过构造函数传递 Repository 和 Service。
- **React**:
    - 使用 Functional Components 和 Hooks。
    - 使用 TypeScript 强类型，尽量避免 `any`。
    - 优先使用 `shadcn/ui` 组件，保持 UI 风格一致。
- **数据库**:
    - 使用 GORM 进行数据库操作。
    - 迁移：使用 GORM AutoMigrate 自动同步 Schema，复杂变更使用 `internal/repository/sqlite/migrations.go` 中的手动迁移机制。

## 常用工作流

需要安装 `task` (Taskfile)。

### 开发 (Dev)
- **全栈开发**: `task dev` (同时启动后端和前端开发服务器)
- **后端开发**: `task dev:backend` (go run)
- **前端开发**: `task dev:frontend` (vite)
- **桌面开发**: `task dev:desktop` (wails dev)

### 构建 (Build)
- **全量构建**: `task build`
- **前端构建**: `task build:frontend`
- **后端构建**: `task build:backend`
- **Docker**: `task docker`
- **桌面构建**: `task build:desktop` (当前平台)

### 检查 (Lint)
- **前端 Lint**: `cd web && pnpm lint`
- **类型检查**: `cd web && pnpm tsc --noEmit`

### 数据库
- **默认**: 使用 SQLite (`~/.config/maxx/maxx.db` 或 `./maxx.db`)。
- **配置**: 开发环境下，可通过环境变量 `MAXX_DSN` 配置 MySQL 或 PostgreSQL 连接字符串，例如 `MAXX_DSN="mysql://user:pass@tcp(localhost:3306)/maxx?parseTime=true"`.

## 注意事项
1.  **前后端通信**: 前端通过 `lib/transport` 与后端通信，该层抽象了 HTTP API 和 Wails Events 两种传输方式。**新增 API 时需同时更新 `interface.ts`、`http-transport.ts` 以及后端对应的 Handler**。
2.  **桌面 vs Web**: 代码需同时兼容 Web 浏览器环境和 Wails 桌面环境。注意不要在通用逻辑中使用仅限浏览器的 API (如 `window` 对象需做检查) 或仅限 Node 的 API。
3.  **API 代理**: 核心逻辑在 `executor` 和 `adapter` 包中。`Router` 负责路由匹配，`Executor` 负责执行请求和重试逻辑，`Adapter` 负责具体的协议转换。
