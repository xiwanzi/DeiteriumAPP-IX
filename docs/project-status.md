# 当前项目状态

2026-09-12 维护收口：启动器管理按本轮授权通过 PR 归入 main，后续 Web/Go 迭代从最新 origin/main 开分支；`.worktrees/dlauncher-admin` 保留为部署追溯入口。首次上线源码为 `fdf788c`。本轮 review 修正同步超时后无法保存终态的问题，未重跑测试、未重新部署；该修正需随下一次获授权的 Go 发布上线。下方“代码仅本地提交”为首次部署时的历史状态。

2026-09-12：**Web 2.0.13 与启动器配套 Go 已上线**，管理导航新增“启动器”，连接同机 McPatch 原生 Web 与 OSS 同步，初始基线已发布。Android 沿用 2.0.12；71 个账号、27 笔订单、5 个委托保留，游戏服务未重启。本轮维护 `.worktrees/dlauncher-admin` / `feat/dlauncher-admin`，代码仅本地提交。见[部署与验证](deployment/dlauncher-0.4-live.md)。下方为较早记录。

2026-09-11：**App 2.0.12（21200）与配套 Go 已于 22:18 上线**，源码 `a0ed4446d074`（PR #23），xiwanzi 更新提醒已推送。注册验证码不再依赖密码，忘记密码浮层接入统一背景模糊；Web 沿用 2.0.10，游戏插件未更换。71 个账号、27 笔订单和 5 个委托全部保留，四个游戏节点当前在线，旧号未注销。维护工作区 `.worktrees/auth-registration-glass`，分支 `xiwanzi/release-v212-live`。见[部署记录](deployment/deuterium-2.0.12-live.md)。下方为较早记录。

2026-09-11：**App 2.0.11（21100）已于 10:40 上线**，源码 `46d4c5e76eb5`（PR #21），xiwanzi 更新提醒已推送；说明为“大幅提升流畅度，提升系统稳定性。”Web/Go 沿用 2.0.10，27 笔订单、4 个委托及原配置保留；三常驻服在线、MEK 离线。维护工作区 `.worktrees/performance-audit-v210`，分支 `xiwanzi/release-v211-live`。见[发布与验证](deployment/deuterium-2.0.11-live.md)。

2026-09-11：**全局过度绘制修正版已在真机覆盖验收**，分支 `xiwanzi/global-overdraw-v210`、工作区 `.worktrees/performance-audit-v210`。四页普通背景退出全屏红色覆盖；68 张固定画面和 4 组真机背景/玻璃对照零差异，同组滑动 GPU 中位数 5.66→4.18 ms。App 仍为 2.0.10（21000），未推送、合并或线上发布。见[验收记录](qa/app-global-overdraw-v210.md)。

2026-09-11：**App 性能优化本机候选**位于 `.worktrees/performance-audit-v210`，分支 `xiwanzi/performance-equivalence-v210`。保持原素材、动画、玻璃和标题折叠，优化计算、持久化与读取，并准备同证书的非 debuggable APK；版本仍为 2.0.10（21000），未推送、合并或部署。见[验收记录](qa/app-performance-equivalence-v210.md)、[全范围审查](reviews/app-performance-full-v210-2026-09-11.md)。真机帧率与手感仍待验证；下方为线上状态。

2026-09-11：**App 2.0.10（21000）、Web 2.0.10 与配套 Go 已于 02:38 上线**，源码 `65c08a290896`（PR #18、#19），已向 xiwanzi 推送更新提醒。优惠券统一应用内通知、停用券/商品可删除、全额退款返券及启动绘制优化已生效；迁移释放 1 条历史退款券占用，26 笔订单与 4 个委托保留。三常驻服在线，MEK 继续停止；游戏插件未更换。维护工作区 `.worktrees/commerce-refunds-performance`，分支 `xiwanzi/release-v210-live`。见[部署记录](deployment/deuterium-2.0.10-live.md)。下方为较早记录。

2026-09-10：**App 2.0.9（20901）、Web 2.0.9、配套 Go/Core 1.0.2、Mail 0.6.2/API3、Bridge 0.6.1 与 XConomy .3 已于 16:36 上线**。主源码 `e4e4c4b7791c`（PR #16），Mail `67db93749bf9`（PR #2）。支持优惠券草稿与批量发放、同批提醒去重、折扣/周期限购及组合交付；22 笔订单、4 个委托和原配置保留，三常驻服恢复，MEK 继续停止禁领。维护工作区 `.worktrees/commerce-v209`，分支 `xiwanzi/release-v209-live`。见[部署记录](deployment/deuterium-2.0.9-live.md)。下方为较早记录。

2026-09-10：**App 2.0.8（20800）、Web 2.0.8 与配套 Go 已于 11:13 上线**，源码 `6566bb617f8d`（PR #14）。默认弧线 D 与双子节图标已内置，管理员可远程切换；线上默认无横幅。22 笔订单、4 个委托及原配置保留。本轮未重跑测试。当前维护工作区 `.worktrees/launcher-icons-v208`，分支 `release/v208-live`。下方为较早记录。 见 [2.0.8 部署记录](deployment/deuterium-2.0.8-live.md)。

2026-09-10：**App 2.0.7（20700）、Web 2.0.7 与配套 Go 已于 03:22 上线**，源码 `4262edd84c03`（PR #12）。Saki 支持逐日递减、最低 5 天差价的升级，严格保留原到期时间；购买确认展示升级实付与淡色删除线原价，不显示公式。空会话才显示输入框上方套餐入口。用户已开启的套餐配置及 11 笔订单、4 个委托保留，三常驻服在线，MEK 离线。当前维护工作区 `.worktrees/saki-upgrade-checkout`，分支 `release/v207-live`。下方为较早发布记录。 见 [2.0.7 部署记录](deployment/deuterium-2.0.7-live.md)。

2026-09-09：**App 2.0.6（20600）、Web 2.0.6 与配套 Go 已上线**。App/Web 源码 `4d9a001ebd0e`（PR #9），Go 最终源码 `515c87fd4936`（PR #10）。账单、Saki/账号后台、头像裁切、购买订单与通知优化见 [2.0.6 部署记录](deployment/deuterium-2.0.6-live.md)。Saki 售卖仍关闭；本轮工作区 `.worktrees/saki-admin-experience`，分支 `release/v206-live`。下方 2.0.5 及更早状态为历史。


此前网站发布（2026-09-09 14:59）：**Web 2.0.5 与配套 Go 已上线**，源码 `0386f1bd5664`。PC 管理、商品 App/裁切预览、游戏邮件配置、头像与购买体验见 [网页 2.0.5 部署记录](deployment/web-2.0.5-live.md) 和 [开发验收](qa/desktop-admin-2026-09-09.md)。

基准：2026-09-09。**App 2.0.5（20500）已发布，历史账单与删除/结果浮窗修复已上线，账号通知已推送**，见 [2.0.5](deployment/deuterium-2.0.5-live.md)。网站与 Go 已随后更新为上述 Web 2.0.5 配套版本，游戏插件仍沿用 2.0.4。

此前部署：App 2.0.4（20401）、Web 2.0.4、配套 Go/Core/XConomy 已发布；账号更新提醒已加入 xiwanzi 通知，原始业务与 Mail/Sync 配置保留。见 [2.0.4 部署记录](deployment/deuterium-2.0.4-live.md)。

| 部分 | 当前状态 | 依据 |
| --- | --- | --- |
| App | 2.0.9（20901），商城修复、我的优惠、合并到账提醒 | [2.0.9](deployment/deuterium-2.0.9-live.md) |
| 网站 | 2.0.9，多物品交付、商品优惠/限购、草稿与批量发券 | [2.0.9](deployment/deuterium-2.0.9-live.md) |
| 后端 | Go e4e4c4b7791c，最优券、限购、购物袋清理、批次与提醒；21 项迁移 | [2.0.9](deployment/deuterium-2.0.9-live.md) |
| 游戏插件 | Core 1.0.2、XConomy .3、Mail 0.6.2/API3、Bridge 0.6.1；三服在线，MEK 离线 | [2.0.9](deployment/deuterium-2.0.9-live.md) |
| 源码维护 | .worktrees/commerce-v209 / xiwanzi/release-v209-live | [2.0.9](deployment/deuterium-2.0.9-live.md) |

## 本次验证与历史证据

2.0.7：构建、资金竞态/集成、安装更新、原生界面、公网产物及真实升级报价验证见 [2.0.7 部署记录](deployment/deuterium-2.0.7-live.md)。

网页 2.0.5：同一源码候选的 58 项 Web 单测、128 项 Go 竞态/集成测试和 40 项隔离浏览器检查通过；上线新验证包括公网 HTML/JS/CSS 摘要、真实管理预览/聊天头像、只读业务接口、三节点重连和临时会话撤销。现有 1 件官方商品为下架状态，生产购买界面未强行上架验收；没有创建真实付款或游戏邮件。

2.0.5：126 项 Android 单测、Lint 0 错误，已发布 2.0.4 到 2.0.5 系统安装器升级、打开与重开、浅深色完整页面回归通过；公网完整 APK、旧版更新与新版不降级、账号通知、后端健康、三节点重连和账单只读验证通过。

2.0.4：121 项 Android 单测、Lint 0 错误；真实系统安装器、取消和覆盖旧版恢复通过。Go 123 项顶层 race/integration、Web 54 项单测/18 项浏览器流程、Core 29、XConomy 10 项数据库测试及 7 种响应契约通过。实际下载/更新检查/业务只读/节点恢复见 [部署记录](deployment/deuterium-2.0.4-live.md)。SMTP 实际邮箱尚未配置。

历史 2.0.3：Android 119 项单测、Lint 0 错误，模拟器交互/Markdown 浅深色/付款时序通过；Go 122 项 race/integration 通过（既有人工夹具跳过），两个实际响应通过契约校验。公网 APK 摘要、更新检查、账号提醒、生产只读接口和三常驻节点在线均已核对。见 [本次 QA](qa/app-v203-2026-09-09.md)。

## 已有验证和仍需验收

历史发布证据包括：App 113 项单元测试及付款四时序模拟器检查；Go 119 项 race/integration 通过（既有人工浏览器夹具跳过）；网站 54 项 Node 测试；Core/Sync/Mail 真实隔离 Youer、保存证明恢复与三常驻服启动验证。这些数目不是本次新跑的测试，整理验证单列于 [交接记录](repository-handoff.md)。

真实手机手感、完整游戏客户端领取/跨服流程以及 MEK 模组兼容仍不能用自动化测试代替。完整第三方 OIDC Provider 仍未包含；账号权限与封禁已在 2.0.6 实现。AI 默认免费 20 次/24 小时、管理员豁免；购买链路已接入，套餐售卖开关仍关闭。

## 数据和版本规则

已存在的用户订单、委托、转账与图片是真实业务，不清空恢复演示。新增迁移必须保留既有 SQL 摘要；版本号、Git 提交、APK/JAR 摘要和线上状态分别记录。外部组件以 [固定版本清单](../components.lock.json) 为准，不跟随默认分支自动升级。

旧入口中的 VIII、Ktor、未实现网站/后端和“付款时序未改”等说明已归入历史语境；不再作为当前结论。
