# AI 会话与流式增量（2026-09-08）

本轮按原 AI v1 接口接入真实模型，保留 Android 现有 AI 页面。用户最新明确覆盖旧计划：免费 `plan_free / free / ProMax` 为 **20 次 / 24 小时**；具有服务端 `platform.admin` 权限的账号不受次数额度限制，仍受并发与超时约束。其余套餐显示 `9999999.00`，暂不开放购买。窗口沿用 Unix epoch 对齐：`epochSeconds - epochSeconds % (windowHours * 3600)`。旧已核实免费值 200/24h 仅为迁移背景，已被最新指令覆盖。当前旧付费权益和最近 7 天额度用量均为 0，不迁移旧聊天正文。

## 权威状态

- `GET /api/v1/ai/me`、`/plans`、`/messages` 保留 v1 JSON 结构；请求和历史始终按鉴权用户与当前会话过滤。
- `POST /api/v1/ai/chat/stream` 接受 `clientMessageId` 和 `content`。相同账号/请求 ID/正文只能启动一次上游；同 ID 不同正文冲突，同 ID 属于旧会话时不得注入当前会话。
- 每个账号同一时间仅一个 AI 生成任务。数据库事务先预留一次额度，成功完成后转为已用；确定失败释放预留。客户端断线不取消已开始任务，服务端在有界截止时间内继续生成并保存正文。
- 进程退出或上游中断导致 UNKNOWN 时，保留已生成正文并明确标记未完成；本窗口预留不退，原 ID 只恢复已有结果，不重新调用模型。活动占用到期后可提交新的请求。与旧“只在完成时扣额度”相比，新增未决额度预留以避免断线绕过限额；恢复时间仍取原计划窗口。
- `/new` 或 `POST /ai/conversation/reset` 创建新会话，不消耗额度。运行中的生成不被重置绕过；旧会话保留审计，普通历史接口不再返回它。

## SSE

输出 `meta`、`status`、`delta {content}`、`sources {sources:[{title,url}]}`、`done {message,quota}`、`error {error}`，并发送保活注释。`meta` 保留会话/用户消息/助手消息 ID 和额度，可带服务器持久化的用户消息。部分正文逐批落库后才向客户端发送；重连从当前持久化文本恢复，完成结果以 `done` 为准。消息增量可带 `status` / `finishReason` 标记截断或未知，不能把 EOF 当完成。管理员额度扩展 `unlimited:true`，客户端显示不限，不能由请求方自报。

模型固定本轮已确认的 DeepSeek V4 Flash，使用官方 Responses API 和内置 `web_search`、`tool_choice:auto`。联网搜索是可用能力，普通问候和无需时效资料的正常回答不强制搜索。只提供这一只读搜索工具，不提供钱包或服务器命令工具。现有真实提示词由受限文件读取，将明确 VIII 品牌引用更新为 IX，追加按需搜索、来源区分和权限边界，不在仓库或交付包复制私有提示词。

仅提取 `response.output_text.delta` 正文与真实 `url_citation` 注释，思考事件不存储、不转发。若供应商未返回 annotation、仅在正文返回 HTTP(S) 链接，则保留该实际链接并标记 `origin:provider_text`，不伪装结构化引用或逐页验证；结构化引用标记 `origin:annotation`。`searchUsed` 只根据真实搜索事件/输出项决定。以 `response.completed` 为正常完成依据；`response.incomplete`、`response.failed` 或连接 EOF 不伪造完成。上游 Responses 为无状态接口，因此数据库保存当前用户历史与恢复状态，每次仅传当前会话受限条数的上下文。依据：[DeepSeek Responses](https://api-docs.deepseek.com/api/create-response/)、[Responses 指南](https://api-docs.deepseek.com/guides/responses_api/)。

## 配置与验证

配置由受限环境文件注入；代码只保存非秘密默认值及可验证边界。免费额度/窗口可配置，默认值来自上述已核实政策。付费购买在 FinanceExecutor 与官方收款路径完成前返回明确不可用，不授予假权益。

入口只需 `DEUTERIUM_AI_CONFIG_FILE` 和 `DEUTERIUM_AI_PROMPT_FILE`：前者兼容当前受限 provider JSON，后者读取 `content` 与 `assistantName`。默认 `reasoningEffort:low`、`maxOutputTokens:4096`、`timeoutSeconds:180`、`maxConcurrent:4`。不得把两个私有配置文件放入仓库或 APK。注册方法为 `s.registerAIV2(mux)`。

验证需包含 fake-provider 的正文/思考分离、断流/长度上限/HTTP 拒绝，隔离 MariaDB 的窗口边界、并发与同 ID 重放、断线继续、旧会话隔离、UNKNOWN 不重复上游，以及已启用后真实 Android SSE 与历史恢复。
