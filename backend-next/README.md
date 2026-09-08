> 当前归档：本目录是已发布 Go 集成后端（含 2.0.1 图片生命周期），并非仅 foundation。部署使用 [当前手册](../docs/deployment/README.md) 与 `ops/` 模板；下文保留开发来源和历史验收。

# Deuterium 后端与 Core 集成

本分支提供 Go 后端、统一身份、Core RPC、钱包及插件侧持久化能力，源码在 `backend-next` 与 [deuterium-core](../deuterium-core/README.md)。App/网页继续沿用 `/api/v1`。**邮箱保持独立插件，Core 只通过公开 API 对接。** Foundation 0.1 是早期验收快照，不再代表当前实现范围。

完整商城/市场/委托与 App/Web/AI 等全栈合并、Linux 运行及生产游戏服操作由 [部署任务](codex://threads/01a07e94-3ac4-7ab2-9d25-2a8698f58cf1) 的 `test-release-v2` 工作线统筹，不能把本分支独立增量与该集成分支混为一份构建。

入口：[总计划](../docs/plans/deuterium-backend-rewrite.md)、[架构决定](../docs/adr/0009-deuterium-backend-core.md)、[各端对接清单](../docs/contracts/backend-integration-checklist.md)、[独立邮箱时序](../docs/contracts/mailbox-integration-v1.md)。

## 已实现

- Go 标准 HTTP 服务、显式 SQL、校验迁移摘要、数据库就绪检查、单实例锁、优雅停止、有界连接与消息读取。
- 旧 app_users 只读导出、JSONL 预检与整批事务导入、冲突阻止、重复导入不覆盖；不导入旧会话和管理权限。
- 旧 Java Argon2i 密码兼容与成功登录升级 Argon2id；游戏名/QQ/账号主体/IP 的持久失败限次、最多 2 个并发哈希验证。
- App Bearer 会话、网页 HttpOnly Cookie、Origin/CSRF、当前用户与注销；会话撤销作用于实时写操作。
- 注册与重置密码：服务器权威 UUID、实际在线玩家私密验证码投递、发送/验证限次、消费一次、重置撤销会话；原始 OTP 在命令完成后清除。
- Core 每节点凭据和固定来源、版本握手、公共聊天入库去重、App 消息向配置节点持久投递、ACK、断线补发及过期控制。
- 物品版本元数据和内容摘要不可变、发布命名空间限制；授权管理员查询节点和物品版本。
- 内置 Core 物品库、不可变版本和归档目录、跨服存储、持续出站事件、独立邮箱创建/查询/取消与持久事件 ACK。
- 钱包查询/收款人解析/实际转账、受控资金幂等查询、排队与未知结果恢复；DIMA/DaoYu 托管保留真实服务账号与资金守恒。Go 不直接写游戏经济库。
- Core、XConomy 和 YouerModSync 适配及隔离服验证，见 [Core 说明](../deuterium-core/README.md)。
- Linux amd64 无 CGO 编译、Windows 本机调试构建、systemd/Nginx 模板。

## 分支与能力边界

本分支侧重身份、Core、钱包与游戏侧能力；本目录已包含商城/市场/委托状态机、资料、私聊、OSS、AI、通知/更新等集成业务。统一第一方 Deuterium ID 已实现，未宣称提供完整第三方 OIDC Provider。旧 OpenAPI 的“existing”描述旧 Ktor，不能直接套用于任一新构建。服务端未接入的能力应明确拒绝，不返回演示成功。

## 构建与验证

使用 Go 1.27，当前验证工具链 1.27.1；锁定依赖见 go.mod/go.sum。运行依赖 MySQL/MariaDB，当前集成验收使用本机隔离 MariaDB 11.8.8。不会内置数据库，也不连接游戏插件的数据库表。

```bash
go test ./...
go vet ./...
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags='-s -w' -o bin/deuterium ./cmd/deuterium
```

真实数据库集成测试用 `go test -race -tags integration ./...`；必须显式设置 DEUTERIUM_TEST_DSN，指向**回环地址、未指定数据库**的隔离 MariaDB 账号，允许临时建库。测试只创建 `deuterium_test_<随机值>` 并清理自身库，拒绝远程数据库；从不读取 DEUTERIUM_DSN 作为测试目标。

可设置 DEUTERIUM_CONTRACT_SAMPLES 为绝对输出目录，随后用 scripts/validate_contract.py 校验实际响应与 App OpenAPI。该脚本需要 PyYAML、openapi-schema-validator、referencing；它们仅为验证工具，不是服务运行依赖。

## 数据库和启动

1. 为新服务创建独立库。迁移账号可执行 DDL；运行账号仅授予本库需要的 SELECT/INSERT/UPDATE/DELETE。不要使用游戏库或 XConomy 账号。
2. 把数据库连接串写入受限环境配置 `DEUTERIUM_DSN`，格式为 `user:password@tcp(host:3306)/deuterium_backend`；连接到非可信网络须启用数据库 TLS。不要提交实际配置。
3. 执行 `deuterium migrate`。迁移前备份；已经执行的 SQL 文件不可修改，后续新增编号迁移。
4. 复制 config.example.json 到私有配置文件，填正式 publicOrigin。当前示例没有节点；可安全启动但不能发送新的 App→游戏消息。
5. 执行 `deuterium serve --config /etc/deuterium/config.json`。`/health/live` 和 `/health/ready` 默认只供回环运维探测。

开发 HTTP 必须 development=true 且监听/Origin 使用回环 IP；生产使用 HTTPS/WSS 反向代理。不要将私有环境配置、token、原始账号导出或测试目录放入交付包。

## 迁移旧账号

```bash
# DEUTERIUM_LEGACY_DSN 使用旧库的只读账号，导出文件不能已存在。
deuterium export-legacy --file /secure/legacy-users.jsonl
# DEUTERIUM_DSN 指向已迁移的新库；默认预检，不写入。
deuterium import-legacy --file /secure/legacy-users.jsonl
deuterium import-legacy --file /secure/legacy-users.jsonl --apply
```

export 只查询 app_users，在只读事务里保留一致快照；不导出明文密码、旧 token、验证码、第三方 client secret 或经济数据。输出文件含敏感哈希，需限定操作系统访问权限；Linux 创建为 0600，Windows 还应使用受限目录 ACL。

必须保留旧 id、server_uuid、current_game_id、qq、password_hash、status 和 UTC created_at/updated_at。当前支持 active/disabled/locked；其他状态拒绝，不自动激活。身份/别名冲突整批失败，绝不按 QQ 或相似名字自动合并。旧 playerRef 不作为长期账号身份；已有第三方 OIDC issuer/sub 的迁移另行核对。

生产旧账号迁移由部署任务记录；本工具的导入范围仍只有账号身份、合法登录别名和密码哈希。旧账单、OIDC client 和其他业务记录不在迁移范围。非数字旧 QQ 可保留为资料，但不成为新 QQ 登录别名。

## Core 节点配置

`deuterium core-key` 为每个节点生成独立随机密钥和摘要。原 token 仅交给对应 Core 私有配置；后端 config.json 的节点仅填写 tokenSha256。

节点包含 id、tokenSha256、chat、itemPrefix、claimEnabled、inventoryDomain、compatibilityProfile、mailCluster、economy 等配置。四服使用 login/amiya/odyssey/mek；实际领取还要求 Mail API、玩家已完成 Sync 加载、当前服务器范围及真实保存屏障全部满足，不能仅凭 claimEnabled 开放。

详情：[Core 协议](../docs/contracts/deuterium-core-v1.md)、[网页协议](../docs/contracts/deuterium-id-web-v1.md)、[增量 OpenAPI](../docs/contracts/openapi-backend-foundation.yaml)。游戏侧发奖没有兼容能力时应拒绝，不使用任意服务器命令作降级方案。

显式授予只读管理权限：`deuterium grant-permission --user <稳定 userId> --permission core.read`。不从旧用户导入、游戏 OP 或 QQ 身份自动授予权限。

## Linux 交付与结果

deploy/deuterium.service 提供专用用户、文件保护和内存预算模板。实际 Linux 主机、证书、配置与运行状态由部署任务维护，不在示例中提交凭据。

测试、协议验证、构建与性能边界见 [本次验收](../docs/qa/backend-foundation-2026-09-08.md)。本机短测不能证明完整业务峰值或 Linux 真正占用，更不能代替与旧服务的同条件对比。
