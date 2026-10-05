# 注册正版 ID 校验本机验收

日期：2026-10-05。工作区 `.worktrees/registration-profile-check`，分支 `xiwanzi/registration-profile-check`，基于 `origin/main` 的 `3a4eaad`。本记录保留本机候选验收证据；后续已按用户授权推送、合并并于 11:26 上线，见[上线记录](../deployment/registration-profile-check-live-20261005.md)。

## 实现及覆盖入口

`backend-next/internal/httpapi/email_identity.go` 的公共注册处理在验证邮箱验证码后、密码计算与数据库创建之前，查询验证码绑定的正版玩家名。复用 `resolveAdmissionProfile` 的 Minecraft 官方接口，服务不可用时拒绝放行。

| 入口 | 最终注册接口 | 提示行为依据 |
| --- | --- | --- |
| 官网、Help 客服中心 | `/api/v1/web/register` | 两站 `account-auth.js` 展示后端 `error.message` |
| 玩家社区 | `/api/v1/web/register` | `AuthScreen.jsx` / `api.js` 展示后端错误 |
| Android App | `/api/v1/account/register` | `BackendApi.kt` / `AuthPages.kt` 展示后端错误 |

不存在：400 `MINECRAFT_PLAYER_NOT_FOUND`，提示核对 Minecraft Java 玩家名并重新获取验证码。超时、限流、异常响应：503 `MINECRAFT_LOOKUP_UNAVAILABLE`，提示稍后重新点击注册。无需前端或 App 升版，无新增迁移。

## 本次验证

Go 1.27.1 / 本机 MariaDB 11.8.8，集成测试仅创建随机测试库并自行清理，不读取生产数据库或发送实际邮件。

- `go test ./...` 与 `go vet ./...` 通过。
- `go test -tags integration ./internal/httpapi -run '^TestRegistrationProfile' -count=1` 通过：两个接口的不存在/不可用拒绝；账号、别名、会话无新增；验证码仍 ACTIVE 且尝试次数未变；同码重试成功；成功仍保持未验证、UUID 未绑定；重放只产生一份账号；无效玩家名、错误验证码及不可信网页来源拒绝。
- `go test -tags integration ./internal/httpapi -run '^(TestRegistrationProfile|TestAdmissionProfile|TestAdmissionPublic|TestMailIdentityGate|TestEmailVerificationLimit)' -count=1` 通过，覆盖白名单申请、游戏身份隔离与验证码并发消费的相关回归。
- 官方档案响应单测覆盖 200、204、404、429、503、无效 JSON、异名、无效 UUID及取消，不将上游故障当作不存在。
- 本机和生产服务器的只读外网检查：官方接口对 `Notch` 返回 200，对本次不存在的测试名称返回 404；生产 Go 当前服务 active，版本仍为 `support-thread-fix-54c389cc9a43`。没有执行生产注册、发验证码或修改账号。
- `git diff --check` 通过。Linux amd64 构建成功，产物 `C:/DeuteriumAPP/deliverables/registration-profile-check/deuterium-linux-amd64`，SHA-256 `7b5cfa2c6ce8e41ef3392a830ee9f6976e9d770ab92e645ef760c174cf1a28fa`。

## 发布范围与限制

已获本轮推送/发布授权并仅发布配套 Go；保留现有官网、Help、社区、App、邮件配置与游戏插件。无数据库结构或业务数据变更。发布时已备份当前 Go 程序与数据库，服务就绪、四节点连接及配置保持检查通过；失败回退只切回原程序，不恢复数据库。玩家名存在不等于账号持有证明，独立游戏验证及通行审批均保留。
