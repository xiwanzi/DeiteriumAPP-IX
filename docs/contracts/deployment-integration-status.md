# 当前部署联调状态与编辑所有权

更新时间：2026-09-08，本轮开发进行中。负责人任务：`01a07db2-e484-7b11-9f0d-3147902063ee`（重写并规划 Deuterium Core 后端）。部署协作任务：`01a07e94-3ac4-7ab2-9d25-2a8698f58cf1`。

## 当前可依赖基线

已提交稳定基线 `8db11097c624a5c7ae4b4f8363a36c9fce3da198`，Foundation 0.1。20 项测试及竞态检查通过；已验证旧 Argon2 账密导入、App/Web 真实登录/注销/当前用户、基础公共聊天、Core 物品版本元数据和 core.read 查询。已有交付包 `delivery/Deuterium-Backend-Foundation-0.1.0.zip`。

本轮第一批 Core/Go 增量已完成构建及数据库集成测试，供部署分支提前合并路由；完整交付仍在继续。**不能把 target 中的中间 JAR 写成完整联调通过**。尚需 XConomy 持久化资金扩展、真实 Sync 保存屏障和游戏内集成验收。

2026-09-08 本机新验证：Go `test -race -tags integration ./...` 全部通过，覆盖并发验证码冷却、5 次错误限制、一次消费、改密撤销会话、真实 HTTP 注册 Cookie 边界、转账幂等与未知状态；Java 16 项测试和 package 通过，独立 Youer 1.21.1 实例成功加载 Core。生产尚未安装此增量。

### 可立即使用的迁移修复（独立提交）

`ff85b25`：保留旧库非数字 QQ 的原显示数据，不创建无效 QQ 登录别名；原游戏名/ID/UUID/密码/状态保持，ImportResult 增加 skippedQqAliases。真实隔离数据库的只读预检、原数据保留、游戏名登录、禁止无效 QQ 登录和重复导入测试通过。该提交可独立 cherry-pick 到部署分支。

只支持导入/导出的独立工具已构建（没有 HTTP serve 命令，避免误部署 WIP 服务）：

- Windows：`C:/DeuteriumAPP/delivery/Deuterium-Identity-Tools-20260908/deuterium-identity-windows-amd64.exe`，SHA-256 `a86f3a1ae3a04cd772b90097b3db4a31a46cb51fac923f120e00a29a55d7e471`。
- Linux：`C:/DeuteriumAPP/delivery/Deuterium-Identity-Tools-20260908/deuterium-identity-linux-amd64`，SHA-256 `50efba8c9f5afa7f8e65aaf078e73e5ed9ae12614081e3e2d857ac14f97048bf`。
- 用法沿用 `export-legacy --file PATH`（DEUTERIUM_LEGACY_DSN）、`import-legacy --file PATH [--apply]`（DEUTERIUM_DSN）。import 默认只读预检；使用已建好的身份 schema，不需要为此应用 Core WIP 迁移。

## 本任务继续持有的编辑范围

- `deuterium-core/**`：完整 `/dc` 插件、内置物品库、WSS、TrChat、玩家目录、邮箱适配、受控经济与同步屏障对接。
- `backend-next/internal/bridge/**`、`internal/config/config.go`、`internal/httpapi/websocket.go`、Core 专属路由/存储和 RPC。
- `backend-next/internal/identity/**` 与本轮新增账号验证码/注册/改密文件；钱包余额、收款人查询、转账、原操作查询与账单必要文件。
- 数据库迁移 **002–009** 留给本任务。部署任务独立分支使用 **010+** 社交/内容/资料迁移，最终合并时一起验证。
- 共享 `internal/httpapi/server.go` 的挂载点由本任务持续修改，部署分支合并时合并路由挂载；不要并发直接改本工作区。

本任务不修改 Android、Web 或独立 Mail 工作区。部署任务的 `test-release-v2` 分支负责其已声明的社交/内容/资料增量；本任务不重复实现私聊、引用等这些端点。

## 当前路由状态

| 路由 | 基线状态 | 本轮 |
| --- | --- | --- |
| POST /account/login；GET /account/me；POST /account/logout | 已实现并实测 | 保持兼容 |
| POST/GET/DELETE /web/session | 已实现 Cookie/Origin/CSRF | 保持兼容；正式 Origin 按部署配置 |
| GET /chat/messages；GET /chat/ws | 基础公共聊天已实现，模拟 Core 客户端验收 | 接真实 Java Core，保留原 chat.send/chat.message 信封 |
| GET /admin/core/nodes；GET /admin/core/items | 基础元数据已实现 | 增加运行能力、目录状态和完整物品元数据 |
| /account/registration-code、register、password-reset-code、password-reset | 基线返回不可用 | 已实现并通过数据库和 HTTP 集成测试；游戏内实测待进行；新增 `/web/register` 返回 HttpOnly 会话 |
| /wallet/balance、balance/refresh、recipients/search、transfers、records | 基线未实现 | 已实现并通过幂等/隔离/未知结果测试；沿用 query/type=auto 和 data.balance/candidates/transfer；真实付款等待 XConomy 持久化 API |
| /bridge/v1/connect | 基础 Core 握手/事件/公共聊天 | 增加命令执行/结果/查询、presence、catalog、独立邮箱事件 |
| 其他社交/内容/资料/商家完整业务 | 非本基线实现 | 部署任务在独立 test-release-v2 分支推进其声明范围 |

## Core 与独立依赖现状

Core 主命令 `/dc`，没有旧物品库存档迁移；已编写 save/get/give/list/info/versions/archive/restore/policy、持久 outbox/journal、WSS 和 Mail 0.6 API 适配。正在补测试与 Go 配套，不交付“只搭骨架”。

邮箱保持独立；Core 不带邮箱 API/实现类入包，不访问邮箱数据库。最新对接已确认 consumerId 必须为 **deuterium-backend**，集群统一为 **deuterium-production**；事件为独立 Mail 的 receipt record，会在 Go 端解析并以集群+eventId 全局去重。

Sync 已找到相关 Codex 任务“修复跨服游戏模式同步Bug”和 `.2` 二进制补丁工程、源码参考。其原 onJoin 会吞错误后报告成功，onQuit 会在保存后清空 capability；**不能直接把 onQuit 当作在线保存 API**。正在实现可验证的屏障适配，不伪造 ready=true。

Core 的 economy 必须显式启用，资金变更只发往配置的经济权威节点；执行前持久化 operationId，未知结果仅查询原记录，不自动重复扣款。普通 XConomy 2.26.3 的 Vault success 可能先于异步 SQL 提交，现已移除该付款路径；只有新增持久化 API 存在时 Core 才公布 economy 能力。扩展会统一旧 pay、Vault 与 Core 写入并验证多服并发。

用户已确认 DIMA 为官方商城收入账号、DaoYu 为真实托管账号。部署任务已报告同名身份无冲突，后续由受控初始化创建零余额系统身份并持久化 UUID；不能注册 App/QQ/密码，普通玩家不得控制。reserve 为付款人→DaoYu，settle 为 DaoYu→冻结的收款人（官方商城为 DIMA），refund 为 DaoYu→原付款人。

## 部署边界

部署任务报告目标 Linux 为独立主机，最终 HTTPS 入口 `https://47.103.99.34`；本任务会提供配置示例，**不把地址或凭据硬编码进程序**。IP 证书、70 个旧账号迁移、实服测试授权和部署由部署任务核对/执行，本任务尚未验证它们的生产完成状态。

四服生产更换和启停由部署任务处理；本任务只构建、测试和提供产物。最终交接需列出 Core、Mail/Bridge/UI、Sync 适配各自版本及依赖，不能仅复制一个 Core JAR 后就宣称跨服领取完整。
