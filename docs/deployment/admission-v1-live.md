# 通行申请与 Velocity Gateway 1.0.0 上线

2026-09-12，Web **2.0.16**、通行申请站及配套 Go 已上线，**22:49:14（UTC+8）** Deuterium Gateway **1.0.0** 在生产 Velocity 4.1.1 / Java 25 启动并启用统一准入。App 保持 2.0.13（21300），按用户要求没有加入 Android 入口或发布新版 APK。

- [申请通行许可](https://47.103.99.34/admission/)
- [白名单管理](https://47.103.99.34/admin?section=whitelist)
- 源码：[PR #28](https://github.com/xiwanzi/DeiteriumAPP-IX/pull/28)、[PR #29](https://github.com/xiwanzi/DeiteriumAPP-IX/pull/29)，最终运行代码对应合并提交 `df9b7dedb0bbca12e1168be6f918c0bca326f322`。维护工作区 `C:/DeiteriumAPP-IX/.worktrees/admission-velocity`。

## 已上线行为

独立申请站保留用户确认的界面、IX 版本标及最后提供的 DSA 方块发射图。玩家无需注册 App，可填写正版玩家名、QQ、可选意向和备注，主动同意公约后提交；取得私密查询凭证，用于查询待审、拒绝原因、通过及后续资格状态。断网未知结果使用原凭证恢复，不盲目创建新申请。

管理后台可查看申请、确认QQ群成员身份及账号使用权后通过、说明原因拒绝、手动添加及移除。资格绑定正版 UUID，申请历史和现行白名单分开保留。移除可选择同时断开在线连接，后台显示代理执行回执；已重新授予的玩家不会被旧指令移除。

代理在正版认证完成、进入子服之前查询准入。**默认入口 Amiya**；已有记录优先返回上次成功游玩的子服，目标不可用时使用 Amiya/Login 备用顺序。自动回退不覆盖原记录，本地原子文件保留跨代理重启的记录。

已导入 **70 个有效旧注册账号**的通行资格，保留 **3 位已有游戏玩家**的历史子服记录。旧代理强制先到 Login 的初始跳转不作为迁移后的游玩目的地。22:52:49 已在真实代理日志中观察到已有玩家直接连接 **MEK**，没有先连接 Login 或 Amiya。

QQ群身份仍由管理员人工核对，不冒充机器人自动验证。后端不可用时拒绝新的登录请求；已有在线连接不会仅因为心跳失败被断开。

## 站点入口与部署过程

申请站目录独立于现有网页：`/var/www/deuterium-admission/current`。最初准备的 9443 端口在本机 HTTPS 验证正常，但公网不可达；因此临时通过标准 HTTPS IP 的 `/admission/` 路径提供入口，现有首页、管理后台及 8443 原监听未改作申请页。新的独立域名尚待用户备案完成后配置，不宣称已经部署域名。

Go 使用 `admissionPublicOrigin=https://47.103.99.34` 与 `admissionApplicationUrl=https://47.103.99.34/admission/` 分别校验来源和生成入口。日后可将 Nginx 站点、证书及这两个地址切换到独立域名，沿用现有申请数据与站点目录。

首次切换因启动器下载索引与较早备份不同而触发自动回退。已确认备份后 22:14 的一次正常 OSS 同步更新了索引，发布版本和内容未变。回退没有恢复数据库；后续在停稳时记录最新索引并验证，保留了用户的新同步结果。029 迁移和旧玩家导入再次执行时为幂等，不重复创建资格或审计。

代理首次安装在设置凭据 ACL 时遇到 Windows 工作组名称映射问题，已自动恢复原进程；改用实际用户 SID 后安装成功。原 6 个代理插件保留，四个子服没有被停止或更换插件。两次操作均有独立备份，未清理玩家存档。

## 实际验证

- 完整 Go `-race -tags integration` 和 vet 通过；新增 7 项数据库集成用例。IP 入口调整后，来源隔离测试与申请相关竞态用例再次通过。
- 代理最终 **13 项 JUnit** 通过，包含在线断开、重授予后的旧指令取消和未知结果保留；同版本真实 Velocity 隔离加载通过。业务代码未因补测改变。
- Web 68 项测试、生产构建通过。浏览器完成实际本机申请/审批/拒绝/移除/手动添加/历史记录及丢失响应恢复；管理页手机深色、桌面浅色与公开页多宽度检查通过。
- 公网网页、脚本、背景、API 和正常 HTTPS 证书验证通过；实际公网浏览器控制台无错误/警告。
- 生产隔离 UUID 探针验证：未知身份拒绝 → 审批后允许 → 撤销后拒绝 → 真实代理返回 `NOT_ONLINE` 断开回执。测试 UUID 不属于真实 Mojang 身份；临时记录、审计及验收会话均已清理，会话再次使用返回 401。
- 代理报告 `ready=true / backend=true / default=amiya / remembered=3`。Login、Amiya、Odyssey、MEK 四个 Core 节点在线。
- 既有 71 条账号记录（含此前注销记录）、27 笔订单、5 个委托及 App 发布清单保留；不复活已注销账号。

真实已有玩家的 MEK 接入已观察到。未使用真实玩家做强制移除试验，也未把隔离/自动化验证描述为所有整合包客户端组合的完整验收。

## 产物与回退

| 产物 | SHA-256 |
|---|---|
| Go Linux amd64 | `68f24e0d5185188789e6a4aeaefa9005782942291c53c99a3e7a0c809ea64f80` |
| Web 2.0.16 ZIP | `68f3f21c51f2e24fa11458a94eb3ec57f1a819d1b0830660938f9cceab5af285` |
| 申请站 1.0.0 ZIP | `35bd030038fb1df55a56c654a78b01ec20cc0c0a8c52b27329571ee16bcc259a` |
| Gateway 1.0.0 JAR | `ec51139dcce4a98670b726f93ecfb1dbb15676592ec7f8cbf28095477137f293` |

当前 Go：`/opt/deuterium/releases/admission-ip-fb12a4d2ece7`。Web：`/var/www/deuterium/releases/web-2.0.16-990a3c9eac3d/dist`。申请站：`/var/www/deuterium-admission/releases/admission-1.0.0-990a3c9eac3d/site`。

Linux 备份位于 `/var/backups/deuterium/admission-v1-990a3c9eac3d/`，IP 调整备份在其 `ip-entry/` 子目录。原 Go/Web 是 `release-2.0.13-15690c138288` / `web-2.0.15-15690c138288/dist`。数据库完整备份 SHA-256：`65a8db39aa4a03a24bb1cb918ebfa49949f607e3264d8e4aad8fc12d4a7e367e`。

代理备份：`E:/Deuterium_IX/deployment-backups/admission-gateway-20260912/` 和 `.../admission-gateway-20260912-retry1/`，保留原路由和玩家记录。原子记录位于 `plugins/deuterium-gateway/last-servers.json`；独立网关凭据只保存在受限配置中，不放入网站、仓库或本文。

需要回退时应协调代理准入与后端版本，避免旧后端不识别准入接口而阻止登录。优先前向修复；不得恢复整库覆盖新申请、订单或注销记录，也不得清空玩家记录。未来改域名时保留现有入口重定向，更新代理提示地址并验证来源校验。

[接口契约](../contracts/admission-v1.md) · [构建及验证清单](artifacts/admission-v1/manifest.json) · [生产探针](artifacts/admission-v1/production-verification.json)
