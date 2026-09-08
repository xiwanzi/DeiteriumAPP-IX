# ADR 0006: AI Chat 架构

## 状态

Accepted

## 背景

DeuteriumAPP 需要同时支持 App AI 聊天和 QQ 群 AI 问答。项目当前已有 Android App、Ktor 后端、Minecraft 插件桥和 MySQL/Flyway 数据层。新功能应尽量复用现有部署，避免新增端口、额外服务和过多运维成本。

## 决策

- AI Chat 作为现有 Backend API 内部模块实现，不拆独立微服务。
- Android App 通过现有公开端口 `28657` 访问 `/api/v1/ai/*`。
- App AI 回复使用同端口 SSE 流式 HTTP，不复用公共聊天 WebSocket。
- NapCat 使用 OneBot v11 反向 WebSocket 连接本地桥端口 `28658` 的 `/bridge/onebot/ws`。
- AI 本地管理后台挂在 `28658` 的 `/admin/ai`，默认只适合本机访问。
- DeepSeek/OpenAI-compatible API Key 只存在后端配置中，App、插件和 NapCat 均不得持有。
- 套餐购买通过新增受控插件桥消息 `wallet.debit.request` 扣当前玩家信用点。
- QQ 入口不做个人记忆、不和 App 玩家身份合并。
- 后端内置薄 AI Gateway：区分普通 API timeout、AI provider timeout 和 SSE heartbeat，不引入独立 LiteLLM/中转站服务。
- App 私聊请求使用 `clientMessageId` 请求交换状态表做幂等，记录 `pending/streaming/completed/failed`，避免断线重试重复扣额度或重复写消息。
- 同一 App 用户同一时间只允许一个 AI 私聊请求处于 `pending/streaming`，用简单串行化避免并发绕过额度和 provider 成本控制。
- AI Gateway 对 provider stream 执行首个正文 delta 超时、chunk 空闲超时、未发送 delta 前一次安全重试，并审计 provider 状态码、首 token 延迟、总耗时、retry 次数和是否已发送 delta。
- 知识库采用受控 agentic RAG：模型最多调用只读 `search_knowledge` 工具 1-2 次，后端执行检索并审计来源；不引入开放式 agent 框架。
- 知识库数据保留 document/chunk 结构，chunk 注入必须限量，不能用知识库内容覆盖系统安全规则。
- 公共聊天 `@客服小祥` 入口固定纯文本、非流式、进程内轻量限频；限频命中只审计并静默丢弃，不向群里反复发送限频提示；固定公共聊天身份为 `客服小祥`，不使用 App AI 私聊名称。

## 取舍

- 选择 SSE 是为了提升 App 响应体感，同时避免维护第二条 App WebSocket。
- 选择后端直连 NapCat 是为了去掉 AstrBot 转发层，减少故障面和延迟。
- 选择数据库动态套餐配置，是为了让 App 购买页实时反映后台配置，不写死 Pro/Ultra 额度。
- 选择后端统一 AI Gateway，是为了集中审计、风控、额度、提示词和模型调用。
- 选择受控查库工具而不是固定摘要注入，是为了让模型只在需要服务器资料时查库，同时保持延迟和成本可控。
- 暂不引入向量库、embedding rerank、多 provider fallback 或完整中转站；这些只有在关键词/chunk 检索无法满足问答质量时再评估。

## 安全边界

- 系统提示词不能作为唯一防线；后端必须做基础提示词攻击拦截。
- 玩家记忆不能改变权限、额度、身份、钱包或套餐状态。
- AI 不执行任意 Minecraft 命令。
- 插件桥扣款只支持受控信用点扣款，不支持通用命令。
- QQ 入口必须使用 token、群白名单、触发规则和熔断开关。
