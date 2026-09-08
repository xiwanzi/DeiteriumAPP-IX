> 历史快照：2026-06-06，不代表当前版本。请先读 [当前状态](../project-status.md)。

# Project Status / 项目状态

Last updated / 更新时间：2026-06-06

## 中文

### 当前版本

- Android：`versionCode 5` / `versionName 1.0.3`
- Backend：`0.1.0`
- Minecraft Plugin：`0.1.0`
- License：Apache License 2.0

### 已实现

- Android 原生 App：账号、钱包、转账、聊天、资料页、本地聊天历史、通知、检查更新。
- 后端：账号、会话、验证码、钱包、转账、聊天、玩家目录、App WebSocket、插件桥 WebSocket、数据库迁移。
- Wiki.js 登录接入：后端提供最小 OIDC Provider，Wiki.js 可通过 Deuterium APP 账号密码登录。
- Minecraft 插件桥：游戏内验证码、聊天互通、Vault/XConomy 钱包操作、钱包监控、服务器事件同步。
- 聊天体验：公共聊天历史、WebSocket 实时消息、在线人数、玩家目录、关心玩家、@ 提及通知。
- 钱包体验：余额刷新、转账、流水展示、离线期间后端记录流水、上线后增量同步。
- 更新检查：后端通过配置判断当前 App 是否为最新版本，不访问数据库。
- 连接稳定性：后端使用标准 WebSocket ping/pong 清理半开连接，降低后台假死后必须重启 App 的概率。
- 后端性能：Repository 写入路径已减少额外查询，验证码尝试计数使用数据库原子自增，App presence 按需读取，过期运营数据定期清理。
- Wiki.js OIDC：新增 `/.well-known/*` 与 `/oauth/*`，OIDC session 与 Android APP session 分离，`/api/v1/*` 不接受 OIDC access token。

### 最近改进

- Android 版本推进到 `1.0.3`。
- 后端配置保持 `app.latestVersionCode=5` / `app.latestVersionName=1.0.3`。
- 新增 backend-only WebSocket 探活热修：
  - `chat.websocketPingIntervalMillis=15000`
  - `chat.websocketTimeoutMillis=35000`
- 新增公开源码导出脚本，避免生产密钥和交付包进入公开仓库。
- 补充中英双语 README、快速开始、技术架构和维护文档。
- 新增后端兼容性性能修复：
  - 余额 upsert 使用单条数据库语句。
  - App presence 首次写入和已存在记录更新都使用单条数据库语句。
  - 钱包流水和转账创建写入后不再读回刚插入的记录。
  - 验证码尝试计数改为数据库原子自增。
  - 登录失败计数改为数据库原子 upsert 递增。
- 新增 V4 维护索引和 6 小时间隔过期数据清理任务。
- 新增 V5 OIDC Provider migration，包含 OIDC client、authorization code、access token 和 browser session 表。
- 已确认 Wiki.js `Deuterium 账号登录` 使用 Generic OIDC，自动加入 `Deuterium Users` 组。

### 待完成

- Android release signing。
- 正式发布包和应用商店/分发渠道策略。
- 将当前 Wiki.js OIDC 内部 HTTP-only 入口迁移为 HTTPS issuer 后，再作为正式公网密码登录入口。
- Android 回到前台时强制检查或重建 WebSocket，实现确定性即时恢复。
- `MainActivity.kt` 按页面拆分，降低 UI 维护成本。
- `AppRepositories.kt` 按账号、钱包、聊天逐步拆分更深的 module interface。
- 更完整的端到端联调脚本和截图/真机验收记录。

## English

### Current Versions

- Android: `versionCode 5` / `versionName 1.0.3`
- Backend: `0.1.0`
- Minecraft Plugin: `0.1.0`
- License: Apache License 2.0

### Implemented

- Native Android App: account, wallet, transfers, chat, profile, local chat history, notifications, update check.
- Backend: account, session, verification, wallet, transfer, chat, player directory, App WebSocket, plugin bridge WebSocket, database migrations.
- Wiki.js login integration: the backend provides a minimal OIDC Provider so Wiki.js can authenticate Deuterium APP accounts.
- Minecraft plugin bridge: in-game verification, chat bridge, Vault/XConomy wallet operations, wallet monitoring, server event sync.
- Chat experience: public chat history, realtime WebSocket messages, online count, player directory, followed players, @ mention notifications.
- Wallet experience: balance refresh, transfers, records, offline server-side record capture, incremental sync after reconnect/login.
- Update check: backend checks the latest App version from config only, without database access.
- Connection stability: backend uses standard WebSocket ping/pong to clean up half-open connections and reduce App restart cases after background/network stalls.
- Backend performance: repository write paths now avoid extra readbacks, verification attempts use atomic database increments, App presence is loaded on demand, and expired operational data is cleaned periodically.
- Wiki.js OIDC: added `/.well-known/*` and `/oauth/*`; OIDC sessions are separate from Android APP sessions, and `/api/v1/*` does not accept OIDC access tokens.

### Recent Improvements

- Android version advanced to `1.0.3`.
- Backend config kept at `app.latestVersionCode=5` / `app.latestVersionName=1.0.3`.
- Added backend-only WebSocket keepalive hotfix:
  - `chat.websocketPingIntervalMillis=15000`
  - `chat.websocketTimeoutMillis=35000`
- Added a clean public source export script to keep production secrets and delivery artifacts out of the public repository.
- Added bilingual README, getting started guide, technical architecture, and maintainer docs.
- Added a backend-compatible performance fix:
  - Balance upsert now uses one database statement.
  - Initial and existing App presence writes now use one database statement.
  - Wallet record and transfer creation no longer read back freshly inserted rows.
  - Verification attempt counting now uses atomic database increments.
  - Login failure counting now uses atomic database upsert increments.
- Added V4 maintenance indexes and a 6-hour expired-data cleanup task.
- Added the V5 OIDC Provider migration for OIDC clients, authorization codes, access tokens, and browser sessions.
- Confirmed Wiki.js `Deuterium 账号登录` uses Generic OIDC and auto-enrolls users into the `Deuterium Users` group.

### Remaining Work

- Android release signing.
- Official release package and distribution strategy.
- Move the current Wiki.js OIDC internal HTTP-only origin to an HTTPS issuer before treating it as a formal public password-login entry point.
- Android foreground WebSocket check/rebuild for deterministic immediate recovery.
- Split `MainActivity.kt` by screen to reduce UI maintenance cost.
- Split `AppRepositories.kt` into deeper account, wallet, and chat module interfaces over time.
- Add more complete end-to-end integration scripts and device acceptance records.
