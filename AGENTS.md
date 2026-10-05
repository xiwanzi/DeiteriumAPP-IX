# Deuterium IX 工程约定

2026-10-05：**注册正版 ID 校验配套 Go 已于 11:26 上线**（PR #68，源码 `6271f468b718`）。官网、Help、社区和 App 的最终注册提交统一查询正版玩家档案，不存在则拒绝注册，查询故障提示重试；验证码不消耗，独立游戏身份验证保留。App 2.0.15 / Web 2.0.25 保持；45 账号、27 订单、5 委托、12 工单、1 注销记录及四节点保留。维护 `.worktrees/registration-profile-check` / `xiwanzi/registration-profile-check`，见[上线记录](docs/deployment/registration-profile-check-live-20261005.md)。下方为较早记录。

2026-10-04：**工单归并修复 Web 2.0.25 与配套 Go 已于 20:52 上线**（PR #66，源码 `54c389cc9a43`）。无引用头回复通过正文编号与同邮箱接续原工单；3 组误拆记录已合并，完整历史保留，已关闭默认移出全部但可搜索，待处理优先。App 2.0.15 保持；41 账号、27 订单、5 委托、1 条已有注销记录保留，工单 15→12。维护 `.worktrees/mail-identity-support` / `xiwanzi/support-threading-fix`，见[上线记录](docs/deployment/support-threading-2.0.25-live.md)。下方为较早记录。

2026-10-04：**邮件主题与数字字体修正已于 20:07 上线**（PR #64，源码 `cf94741842ec`）。工单号仅留正文，不再写入邮件主题；工单号恢复 Consolas，普通数字恢复 Arial，中文和排版不变。App 2.0.15 / Web 2.0.24 保持。维护 `.worktrees/mail-identity-support` / `xiwanzi/mail-subject-digits`，见[上线记录](docs/deployment/mail-subject-digits-live.md)。下方为较早记录。

2026-10-04：**Web 2.0.24 与配套 Go 已于 19:56 上线**（PR #62，源码 `0a6fea8d047c`）。工作台支持主动建单、首封带编号人工邮件且不发自动回执；超级管理员可修改其他账号游戏 ID，保留原游戏身份及资产。App 2.0.15 不变；本轮基线 40 账号、6 工单、27 订单、5 委托、1 注销记录保留，四节点在线。维护 `.worktrees/mail-identity-support` / `xiwanzi/outbound-support-ticket`，见[上线记录](docs/deployment/outbound-support-2.0.24-live.md)。下方为较早记录。

2026-10-04：**Web 2.0.23 与配套 Go 已于 14:09 上线**（PR #60，源码 `77af63a1b290`）。自动工单回执、人工回复及相关网页称呼统一为“Deuterium IX 玩家支持”；邮箱地址、样式和配置保持。App 2.0.15 不变，8 账号、4 工单、27 订单、5 委托保留。维护 `.worktrees/mail-identity-support` / `xiwanzi/player-support-name`，见[上线记录](docs/deployment/player-support-name-live.md)。下方为较早记录。

2026-10-04：**工单工作区 Web 2.0.22 与配套 Go 已于 13:46 上线**（PR #58，源码 `939d0b780f77`）。默认收起全站导航，铺满可用宽高，固定回复区，前置处理操作；草稿/重试与中文搜索修复。App 2.0.15、邮件样式和配置保持；8 账号、3 工单、27 订单、5 委托、0 注销记录保留，四节点在线。维护 `.worktrees/mail-identity-support` / `xiwanzi/support-workspace-efficiency`，见[上线记录](docs/deployment/support-workspace-2.0.22-live.md)。下方为较早记录。

2026-10-04：**邮件系统已接通，最终 Go 修正于 12:49 上线**（PR #55、#56，源码 `b0b6686cf1ad`）。DM、客服 SMTP/IMAP、内部邮箱已启用并完成真实联调；品牌图改用 HTTPS 外链，回执去掉问题摘要，按用户要求保留署名/页脚布局与 Gmail 折叠行为。App 2.0.15 / Web 2.0.21 保持；原 7 账号保留，启用后新增 1 个邮箱注册账号，当前 8 账号、27 订单、5 委托、0 注销记录。维护 `.worktrees/mail-identity-support` / `xiwanzi/mail-live`，见[上线记录](docs/deployment/mail-system-live-20261004.md)。下方候选及未接通说明均为历史状态。

## 邮件视觉约定（2026-10-04）

所有邮件复用[邮件品牌与字体规范](docs/contracts/mail-visual-standard.md)：正文 HarmonyOS Sans SC，数字 Novecento Sans Wide（六位验证码保留原 Consolas 等宽字体），字标沿官网页脚字形。徽章 48px，联系文字 12px、版权 11px；不能再以缩小品牌与页脚解决重心。自定义字体与回退都须按实际渲染验收，不宣称所有邮箱客户端均支持。字体与邮件样式已上线，运行状态以上方最新记录为准。

2026-10-04：**App 2.0.15（21500）、社区 Web 2.0.21 与配套 Go 已于 09:30 上线**（PR #53，源码 `6d5a2fdde8f1`）。新增邮箱注册、独立游戏验证、中文昵称、邮件系统及邮件工单，旧账号保持已验证。7 账号、27 订单、5 委托和 0 注销记录保留，四个 Core 节点在线；游戏插件未更换。新 DM / 企业邮箱授权码尚待后台配置，邮箱注册及工单收发暂未启用；旧审核邮件沿用原 SMTP。维护 `.worktrees/mail-identity-support` / `xiwanzi/release-v215-live`，见[上线记录](docs/deployment/deuterium-2.0.15-live.md)。下方未部署说明为早期候选记录。

2026-10-04：**按用户确认名单，00:08 已物理清除其余 65 条 App 账号，仅保留七个 active 账号，注销记录为 0**。关联 1 笔 App 转账、2 笔取消未付款订单和 3 个所属图片对象已清理；其余 27 订单、5 委托及游戏余额/存档/白名单保持。见[清理记录](docs/deployment/account-purge-20261004.md)，不得从旧备份自动恢复。注册/找回/改密验证码模板及邮件卡片改动仅为本机候选，未上线，见[模板验收](docs/qa/verification-mail-templates-20261004.md)。


2026-10-03：**邮件系统、独立游戏验证与邮件工单本机候选已完成，未推送/部署**。工作区 `.worktrees/mail-identity-support`、分支 `xiwanzi/mail-identity-support`。邮箱注册后再补 QQ/游戏验证码；旧账号默认已验证，中文昵称与游戏 ID 分离。工单按客服邮箱收件生成日期＋四位随机编号，DM 自动回执，企业邮箱人工回复；没有玩家在线工单对话。原管理员为超级管理员，新管理员仅工单。真实 SMTP/IMAP 授权码及真机验收待完成，App/Web 发布版本未变。见[本机验收](docs/qa/mail-identity-support.md)。下方为既有发布记录。


2026-09-28：**Web 2.0.20 与配套 Go 已上线**（PR #50、#51）。后台新增「模型管理」，统一栏目、界面、Deuterium ID 登录和管理员权限，复用原 Velocity API；256 模型、476 文件、模型 revision 11 / 配置 revision 3 与四节点 READY 保留。App 仍为 2.0.14；72 条账号、29 笔订单、5 个委托及已有注销记录保留，游戏插件未更换。维护 `.worktrees/chat-domain` / `xiwanzi/model-management-record`，见[上线记录](docs/deployment/model-management-live.md)。下方为较早状态。

2026-09-28：**YSM 统一管理本机候选已完成，未推送、未合并、未部署**。复用 `.worktrees/chat-domain`，分支 `xiwanzi/ysm-admin-integration`，基于主线 `df8f8df`。后台新增 YSM 模型栏目，统一界面、登录与管理员权限，通过受限 Go 适配现有 Velocity API。现有模型/配置权威和游戏插件保留，生产隧道及 Web/Go 发布待本轮授权。见[本机验收](docs/qa/ysm-admin-integration.md)与[候选上线步骤](docs/deployment/ysm-admin-candidate.md)。下方仍为现有上线版本。

2026-09-26：**App 2.0.14（21400）与配套 Go 已于 20:00 上线**（PR #48，源码 `4e95ae0df69b`）。修复公共聊天在线名单及键盘收起期间快速返回的回调空档。Web 保持 2.0.19，游戏插件未更换；72 条账号记录、29 笔订单、5 个委托及 1 条已有注销记录保留。维护 `.worktrees/chat-domain` / `xiwanzi/release-v214-live`，见[部署记录](docs/deployment/deuterium-2.0.14-live.md)。下方“尚未部署”为本轮早期候选记录。

2026-09-26：**聊天在线名单与连续返回修复已完成本机实现，尚未推送/部署**。从最新主线 `cbc522d14ba5` 建立 `xiwanzi/chat-online-back-fix`，复用 `.worktrees/chat-domain`；Android 基于当前线上 2.0.13（21300）且暂不升版。在线名单需配套 Go 接口，返回修复已通过模拟器事件回归，厂商侧滑仍待真机验收。见[本机验收](docs/qa/chat-online-back-fix.md)。下方为既有上线记录。

2026-09-19：**App 宣传页 https://app.deuteriumix.com 已于 21:36 上线**。沿用第七版设计，删除未接正式服务等旧说明；下载和网页入口均为“敬请期待”，本域名不提供 APK。独立静态站与 HTTPS 自动续期验证通过，见[上线记录](docs/deployment/app-landing-live.md)。

2026-09-19：**玩家社区域名 https://chat.deuteriumix.com 已于 21:12 上线**。独立 HTTPS 证书和自动续期验证通过，Go 精确兼容新旧来源，App 2.0.13 / Web 2.0.19 保持；旧 IP 接口和启动器入口保留。源码 `a1064819ac0a`，维护 `.worktrees/chat-domain`，见[上线记录](docs/deployment/chat-domain-live.md)。下方为较早记录。

2026-09-15：**Go 并发优化已于 20:47:45 上线**（PR #43，源码 `a824088cb518`）。请求分组限额、公共聊天读取合并、AI/密码短时等待与管理员容量接口已部署。App 2.0.13、Web 2.0.19、启动器 0.6.1、申请站 1.0.2 保持；配置、71 条账号记录、27 笔订单和 5 个委托保留。按用户要求未运行测试或压测。维护工作区 `C:/DeiteriumAPP-IX/.worktrees/backend-concurrency`，见[上线记录](docs/deployment/backend-concurrency-live.md)。下方为较早记录。

2026-09-13：**申请站 1.0.2 配色优化已于 20:48 上线**（PR #41）。主页面减少装饰性亮黄，弹窗查询改深色按钮、返回改描边按钮；仅替换静态 CSS，申请、审核、邮件和其他组件不变。见[上线记录](docs/deployment/admission-1.0.2-live.md)。下方为较早记录。

2026-09-13：**Web 2.0.19、配套 Go 与启动器 0.6.1 已上线**（PR #39）。管理页新增程序签名发布，继续使用现有 OSS；启动器内嵌玩家社区并接入独立更新器，实际自动升级通过。App 保持 2.0.13，业务与白名单状态保留。见[上线记录](docs/deployment/dlauncher-0.6.1-live.md)。下方为较早记录。

2026-09-13：**审核邮件配色保护与小尺寸徽章已于 10:47 上线**（PR #37）。通过/拒绝模板均采用已确认的锁色处理，内嵌图标从 1.82 MB 降至 32.5 KB，仍使用 CID 附件。SMTP 配置及审批流程保留，Web 仍为 2.0.18、App 2.0.13。见[上线记录](docs/deployment/mail-appearance-live.md)。下方为较早记录。

2026-09-13：**Web 2.0.18、配套 Go 与白名单审核邮件已于 02:05 上线**（PR #35）。通过/拒绝结果自动发至申请人的 QQ 邮箱，使用后台已保存的 SMTP 配置；两种模板实际测试均被接收。支持模板预览、测试、投递状态与失败重试，过期通知取消。App 仍为 2.0.13，70 条已撤销资格未恢复。见[上线记录](docs/deployment/admission-review-email-live.md)。下方为较早记录。

2026-09-13：**SMTP 队列修复已于 00:36 上线**，解决 MariaDB 10.5 不支持 `SKIP LOCKED` 导致测试邮件始终待发送的问题。两封已排队测试邮件已被 QQ SMTP 接收，配置/授权码保留。Web 仍为 2.0.17，App 2.0.13，70 条已撤销白名单资格未恢复；审批通过邮件尚待正文确定。见[上线记录](docs/deployment/smtp-queue-fix-live.md)。下方为较早记录。

2026-09-13：**Web 2.0.17 已于 00:10 上线**，修复白名单列表切回申请记录导致全屏白屏。按用户最新要求，00:14 已撤销全部 70 个现有白名单资格，当时有效资格为 0；账号及历史保留，在线连接未主动断开。以后须重新申请审批，禁止自动恢复旧导入资格。App、Go、Gateway 沿用原版。见[部署与重置记录](docs/deployment/web-2.0.17-live.md)。下方为较早记录。

2026-09-12：**通行申请站、Web 2.0.16、配套 Go 与 Velocity Gateway 1.0.0 已上线**。申请入口 `https://47.103.99.34/admission/`，管理入口 `/admin?section=whitelist`；默认进 Amiya，重连优先上次子服。70 个有效旧账号保留通行资格。App 保持 2.0.13，本轮仅独立网页。维护工作区 `C:/DeiteriumAPP-IX/.worktrees/admission-velocity`，先读[部署记录](docs/deployment/admission-v1-live.md)。下方为较早记录。

2026-09-12：**App 2.0.13（21300）、Web 2.0.15 与配套 Go 已于 19:15 上线**，源码 `15690c138288`（PR #26），xiwanzi 更新提醒已推送。工作台支持当前管理员密码确认的永久注销，App/Web 同步清理关联展示，历史交易匿名保留；已部署启动器 0.5 源码一并合入，内容版本 5 和配置未改。71 个账号、27 笔订单、5 个委托保留，四节点在线；本轮未注销真实账号。维护工作区 `.worktrees/account-erasure`，分支 `xiwanzi/release-v213-live`。见[部署记录](docs/deployment/deuterium-2.0.13-live.md)。下方为较早记录。

2026-09-12：**管理员永久注销账号候选已完成本机实现与验收，尚未推送/部署**。源码 `038718b068fb`，工作区 `.worktrees/account-erasure`、分支 `xiwanzi/account-erasure`。工作台校验当前管理员密码，后端清理账号与展示数据，App/Web 同步清理缓存，历史交易匿名保留。没有操作真实旧号；线上仍为 2.0.12。见[验收记录](docs/qa/account-permanent-deletion.md)。

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
