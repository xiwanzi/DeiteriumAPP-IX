# Deuterium IX 工程约定

2026-09-12：**DLauncher 0.5.0 / Web 2.0.14 配套 Go 已上线**，切换为整包发布后仅 McPatch 更新，删除首次安装接口和配置；Deuterium IX 使用 D9 徽章及 DSA 素材，其他入口隐藏。本轮从 origin/main 4a9c249 继续，保留并上线同步超时终态修复。维护 `.worktrees/dlauncher-bundled` / `feat/dlauncher-bundled`；用户要求本轮不生成完整游戏包。见[部署记录](docs/deployment/dlauncher-0.5-live.md)。

2026-09-12 维护收口：启动器管理按本轮授权通过 PR 归入 main，后续 Web/Go 迭代从最新 origin/main 开分支；`.worktrees/dlauncher-admin` 保留为部署追溯入口。首次上线源码为 `fdf788c`。本轮 review 修正同步超时后无法保存终态的问题，未重跑测试、未重新部署；该修正需随下一次获授权的 Go 发布上线。下方“代码仅本地提交”为首次部署时的历史状态。

2026-09-12：**Web 2.0.13 与启动器配套 Go 已上线**，管理导航新增“启动器”，连接同机 McPatch 原生 Web 与 OSS 同步，初始基线已发布。Android 沿用 2.0.12；71 个账号、27 笔订单、5 个委托保留，游戏服务未重启。本轮维护 `.worktrees/dlauncher-admin` / `feat/dlauncher-admin`，代码仅本地提交。见[部署与验证](docs/deployment/dlauncher-0.4-live.md)。下方为较早记录。

2026-09-11：**App 2.0.12（21200）与配套 Go 已于 22:18 上线**，源码 `a0ed4446d074`（PR #23），xiwanzi 更新提醒已推送。注册验证码不再依赖密码，忘记密码浮层接入统一背景模糊；Web 沿用 2.0.10，游戏插件未更换。71 个账号、27 笔订单和 5 个委托全部保留，四个游戏节点当前在线，旧号未注销。维护工作区 `.worktrees/auth-registration-glass`，分支 `xiwanzi/release-v212-live`。见[部署记录](docs/deployment/deuterium-2.0.12-live.md)。下方为较早记录。

2026-09-11：**App 2.0.11（21100）已于 10:40 发布**，源码 `46d4c5e76eb5`（PR #21），xiwanzi 唯一更新提醒已推送。更新说明：“大幅提升流畅度，提升系统稳定性。”采用同签名非 debuggable 性能包；Web/Go 沿用 2.0.10，游戏插件未更换，27 笔订单和 4 个委托保留。维护工作区 `.worktrees/performance-audit-v210`，分支 `xiwanzi/release-v211-live`。见[部署记录](docs/deployment/deuterium-2.0.11-live.md)。下方“未发布”为此前候选阶段记录。

2026-09-11：**全局过度绘制修正版已完成真机覆盖验收**。维护工作区 `.worktrees/performance-audit-v210`，分支 `xiwanzi/global-overdraw-v210`，App 仍为 2.0.10（21000）。普通背景改为缓存复用，玻璃保留原采样源；四页背景退出全屏红色覆盖，68 张固定画面及 4 组真机对照零像素差异。见[验收记录](docs/qa/app-global-overdraw-v210.md)。尚未推送、合并或线上发布。

2026-09-11：**本机 App 性能优化候选**维护于 `.worktrees/performance-audit-v210`，分支 `xiwanzi/performance-equivalence-v210`，版本仍为 2.0.10（21000）。保留原视觉/动画参数，完成列表计算、请求持久化、图片解析、绘制和非 debuggable 构建优化；实际验证和限制见[验收记录](docs/qa/app-performance-equivalence-v210.md)。尚未推送、合并或部署，线上状态仍以下方 02:38 发布记录为准。

2026-09-11：**App 2.0.10（21000）、Web 2.0.10 与配套 Go 已于 02:38 上线**，源码 `65c08a290896`（PR #18、#19），已向 xiwanzi 推送更新提醒。优惠券统一应用内通知、停用券/商品可删除、全额退款返券及启动绘制优化已生效；迁移释放 1 条历史退款券占用，26 笔订单与 4 个委托保留。三常驻服在线，MEK 继续停止；游戏插件未更换。维护工作区 `.worktrees/commerce-refunds-performance`，分支 `xiwanzi/release-v210-live`。见[部署记录](docs/deployment/deuterium-2.0.10-live.md)。下方为较早记录。

2026-09-11：优惠券生命周期、商品删除及启动绘制修正已通过 [PR #18](https://github.com/xiwanzi/DeiteriumAPP-IX/pull/18) 合入 main，源码 `303db4ee57f9`。App/Web 2.0.10（App 21000）准备测试交付；线上部署状态见[2.0.10 交付记录](docs/deployment/deuterium-2.0.10-handoff.md)。下方“尚未推送、合并”是开发阶段记录。

2026-09-11：本机已完成优惠券应用内通知、停用/草稿券删除、商店商品删除、全额退款返券与启动/加载绘制优化。维护工作区 `.worktrees/commerce-refunds-performance`，分支 `xiwanzi/commerce-refunds-performance`；[验收记录](docs/qa/commerce-refunds-performance.md)。Go 全量竞态/集成、Android 134、Web 65 及原生通知检查通过；尚未推送、合并或部署，线上仍为下述 2.0.9。

2026-09-10：**App 2.0.9（20901）、Web 2.0.9、配套 Go/Core 1.0.2、Mail 0.6.2/API3、Bridge 0.6.1 与 XConomy .3 已于 16:36 上线**。主源码 `e4e4c4b7791c`（PR #16），Mail `67db93749bf9`（PR #2）。支持优惠券草稿与批量发放、同批提醒去重、折扣/周期限购及组合交付；22 笔订单、4 个委托和原配置保留，三常驻服恢复，MEK 继续停止禁领。维护工作区 `.worktrees/commerce-v209`，分支 `xiwanzi/release-v209-live`。见[部署记录](docs/deployment/deuterium-2.0.9-live.md)。下方为较早记录。

2026-09-10：**App 2.0.8（20800）、Web 2.0.8 与配套 Go 已于 11:13 上线**，源码 `6566bb617f8d`（PR #14）。默认弧线 D 与双子节图标已内置，管理员可远程切换；线上默认无横幅。22 笔订单、4 个委托及原配置保留。本轮未重跑测试。当前维护工作区 `.worktrees/launcher-icons-v208`，分支 `release/v208-live`。下方为较早记录。 见 [2.0.8 部署记录](docs/deployment/deuterium-2.0.8-live.md)。

2026-09-10：**App 2.0.7（20700）、Web 2.0.7 与配套 Go 已于 03:22 上线**，源码 `4262edd84c03`（PR #12）。Saki 支持逐日递减、最低 5 天差价的升级，严格保留原到期时间；购买确认展示升级实付与淡色删除线原价，不显示公式。空会话才显示输入框上方套餐入口。用户已开启的套餐配置及 11 笔订单、4 个委托保留，三常驻服在线，MEK 离线。当前维护工作区 `.worktrees/saki-upgrade-checkout`，分支 `release/v207-live`。下方为较早发布记录。 见 [2.0.7 部署记录](docs/deployment/deuterium-2.0.7-live.md)。

2026-09-09：**App 2.0.6（20600）、Web 2.0.6 与配套 Go 已上线**。App/Web 源码 `4d9a001ebd0e`（PR #9），Go 最终源码 `515c87fd4936`（PR #10）。账单、Saki/账号后台、头像裁切、购买订单与通知优化见 [2.0.6 部署记录](docs/deployment/deuterium-2.0.6-live.md)。Saki 售卖仍关闭；本轮工作区 `.worktrees/saki-admin-experience`，分支 `release/v206-live`。下方 2.0.5 及更早状态为历史。

当前网站发布（2026-09-09 14:59）：**Web 2.0.5 与配套 Go 已上线**，源码通过 PR #7 合入 main（`0386f1bd5664`），见 [网页 2.0.5 部署记录](docs/deployment/web-2.0.5-live.md)。本轮维护工作区 `C:/DeiteriumAPP-IX/.worktrees/desktop-admin`，部署文档分支 `release/web205`。PC 管理导航/表格、商品手机与裁切预览、游戏邮件文案、玩家头像和购买体验已发布。App 保持已发布 2.0.5（20500），游戏插件沿用 2.0.4 配套版本；下方“网站和 Go 沿用 2.0.4”是 App 发布时的历史状态。

当前 App 发布（2026-09-09）：**2.0.5（20500）已上线并推送账号提醒**，源码 `bcfe87e398e75`，见 [2.0.5 部署记录](docs/deployment/deuterium-2.0.5-live.md)。修复历史账单时钟偏差、删除入口及确认浮窗、退款成功居中；当前维护工作区仍为 `C:/DeiteriumAPP-IX/.worktrees/release-v204`，分支 `release/v205`。网站、Go 程序和游戏插件沿用 2.0.4。

此前 2.0.4 发布（2026-09-09）：**App 2.0.4（20401）、Web 2.0.4、配套 Go、Core 1.0.1 和 XConomy .2 已上线**。源码主体 `812f1d1a5056`，见 [2.0.4 部署记录](docs/deployment/deuterium-2.0.4-live.md)。当前维护工作区为 `C:/DeiteriumAPP-IX/.worktrees/release-v204`；用户已授权本轮公开仓库推送与部署。SMTP 页面与密钥已就绪，真实邮箱参数尚待管理员填写。

2.0.4 已修复安装后的旧任务残留及加载等待，统一游戏/App 流水和日统计、最新记录排序、删除按钮与主题切换；网页支持永久删除公告、介入邮件队列和完整交易审计。2.0.3 与 20400 候选记录是历史。

先读 `CONTEXT.md`、`docs/project-status.md`、`docs/prd/deuterium-current.md` 和目标模块说明；运行 `git status --short`。本仓库是 2026-09-08 从多工作区整理的统一维护入口，旧工作区不得自动覆盖回来。

- Android 当前模块为 `android-app/ui-lab`；Web 为 `web-app`；Go 为 `backend-next`；游戏插件为 `deuterium-core`。`android-app/app` 是保留的旧模块，不作为现行发布入口。
- Mail 与 YouerModSync 在各自独立私有仓库维护。先核对 `components.lock.json`，接口更改必须同步配套版本和文档，禁止复制出第二份业务权威。
- 用户最新明确要求优先于历史说明。历史 PRD/QA 有适用日期，不能把旧 Ktor/VIII/演示状态当作现状。
- Android → Go API → Core → 独立 Mail / XConomy / Sync。App/Web 不直连数据库或持有服务器秘密。UUID 由游戏服务权威解析，QQ/游戏名不替代资产身份。
- 资金、库存、退款、领取、会话变更必须验证鉴权、幂等、并发、未知结果及恢复。未知结果不自动换请求键、不重复发奖，不用查不到记录证明可以退款。
- 不修改已执行迁移；新增迁移需备份与回退说明。不清除真实用户测试订单、委托或图片作为种子数据。
- 保持四入口商城/市场/信息/我的及现有主题、玻璃和动效设计；不恢复早期演示导航、模拟登录或彩色主题。
- 沿用相邻实现，不做无关重构。文档默认中文，提交采用英文 Conventional Commits。
- 大改动走独立分支和 PR；推送、合并与部署依本轮授权。不得提交私钥、token、密码、运行时、数据库或本机绝对路径配置；不得 `git add .`。
- 按修改范围运行构建/测试；纯文档检查链接、事实和差异即可。历史测试不冒充本次测试，模拟器不代替真机及完整游戏切服验收。
- 完成时说明修改、验证、提交合并状态及真实部署状态。后续发布继续按对应轮次授权执行。
