# AI Chat v1 实施计划

## 1. 本次目标

实现 App + QQ 群 AI Chat v1 的端到端基础闭环：App 可聊天、可看额度、可购买套餐；后端可审计、可限额、可调用模型；插件桥可扣信用点；QQ 可通过 NapCat 进入后端。

## 2. 涉及端

- Android：新增 AI 底部栏、SSE 流式解析、套餐购买入口。
- 后端：新增 AI 数据表、公开 API、SSE、DeepSeek/OpenAI-compatible client、审计、限额、购买订单、OneBot adapter、本地后台。
- Minecraft 插件：新增 `wallet.debit.request/result` 受控扣款能力。
- 文档：更新统一接口契约、OpenAPI、PRD、ADR 和 plan。

## 3. 数据流

App AI 聊天：

```text
Android -> POST /api/v1/ai/chat/stream
Backend -> clientMessageId 请求状态表/限额/风控/上下文/记忆
Backend -> 可选受控 search_knowledge 查库，返回 sources
Backend -> DeepSeek compatible API
Backend -> SSE meta/status/sources/delta/done/error + heartbeat comment
Backend -> 写消息、请求交换状态和可诊断审计
```

套餐购买：

```text
Android -> POST /api/v1/ai/purchases
Backend -> 校验 plan/clientRequestId
Backend -> wallet.debit.request
Minecraft Plugin -> Vault/XConomy 扣款
Backend -> grant entitlement + wallet record
Android -> 展示新额度
```

QQ 群：

```text
QQ -> NapCat -> /bridge/onebot/ws
Backend -> 群白名单/触发规则/去重/审计
Backend -> send_group_msg
```

## 4. 失败场景

- AI 未启用或 Key 缺失：返回 `AI_DISABLED`。
- 模型超时或不可用：首个正文 delta 超时、连续正文 delta 空闲超时、provider HTTP 错误都返回 `AI_PROVIDER_TIMEOUT` / `AI_PROVIDER_UNAVAILABLE`，并写入延迟、retry、是否已发送 delta；provider keep-alive 不无限延长首个正文 delta 或正文 delta 空闲等待。
- DeepSeek 429/500/503 等 provider 错误：审计记录原始 HTTP 状态码和错误摘要，对 App 统一返回可展示的 AI 临时不可用提示。
- App 断流或重试同一 `clientMessageId`：后端不得重复写用户消息或重复扣额度；已有完整回复时返回既有回复；仍在处理中且未超时时提示稍后刷新；处理中超时或已失败时提示重新发送新消息；已收到部分 delta 后断开可消耗一次额度并审计。
- 同一用户同一时间只允许一个 AI 私聊请求处于 `pending/streaming`，防止并发请求绕过额度检查。
- 知识库无命中或 tool decision 失败：降级为普通 AI 回复，不阻塞整条消息。
- 额度耗尽：返回 `AI_QUOTA_EXCEEDED` 和恢复时间。
- 购买余额不足：返回 `BALANCE_INSUFFICIENT`。
- 购买结果未知：返回 `AI_PURCHASE_RESULT_UNKNOWN`，不授予权益。
- QQ WebSocket 鉴权失败：关闭连接。

## 5. 验收

- 后端测试通过。
- 插件构建通过。
- Android 单元测试通过。
- App AI tab 能编译进入主界面。
- App AI 私聊发送后立即显示用户气泡并清空输入框；回复中输入框可编辑，仅发送按钮禁用。
- App AI 首次同步未完成时不允许发送，避免历史刷新覆盖本地草稿。
- App AI 私聊流式阶段纯文本增量显示，`done` 后再渲染 Markdown；公共聊天不使用 Markdown、不流式。
- 公共聊天 `@客服小祥` 入口有轻量进程内限频和单群进行中上限；限频/进行中/AI 错误命中只审计并静默丢弃，避免 QQ/NapCat 反向刷屏；回复固定纯文本、固定公共聊天身份。
- 契约文档列明新增 API、SSE 事件、AI 幂等规则和插件桥消息。
