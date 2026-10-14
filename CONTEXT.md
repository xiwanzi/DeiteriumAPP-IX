2026-10-15：**启动器 0.6.10 于03:00发布，Web 2.0.41 与配套 Go 于02:58上线，03:03公网核验通过**（DeLauncher PR #11 / 主 PR #132）。共享更新运行库并支持签名文件级增量，更新器73.80→2.72 MiB，完整包271.75→201.10 MiB；更新器样式保持。首次0.6.9迁移仍需完整包。原账号、业务、配置及游戏资源0.0.60-dev保留，App2.0.29保持，三常驻节点在线、MEK停止，本轮未重启游戏服。维护 `.worktrees/launcher-incremental` / `xiwanzi/launcher-incremental`。见[上线记录](docs/deployment/launcher-incremental-0.6.10-live.md)。下方为历史状态。

# Deuterium IX 项目上下文

2026-10-14：**App 2.0.29（22900）、Web 2.0.40 与配套 Go 已于14:19上线，14:21公网核验通过**（PR #130，源码 `a8377b1387d1`）。上线官方资料查询与登录容量优化、App 长期/Web 30 天会话、设备管理和未验证身份占用处理；未验证账号不自动清理，已验证身份不换绑。61 条账号记录、47 订单、5 委托、3,085 在售商品、12 工单、2 注销记录、8 个已验证身份及原配置保留，105 张原表指纹核对一致。三常驻节点在线、MEK停止；启动器0.6.8/资源0.0.60-dev保持，本轮未重启游戏服。维护 `.worktrees/account-experience-capacity` / `xiwanzi/account-experience-capacity`，见[上线记录](docs/deployment/account-experience-2.0.29-live.md)。下方候选说明为历史状态。

2026-10-13：**资源0.0.59-dev于17:03发布，17:04公网核验通过**（DeLauncher PR #6）。修复进服设置页关闭后重新黑屏渐出的遗漏流程，正常首次过渡保留；仅更新主开屏JAR及显示版本。App2.0.27、聊天0.1.10、路标配置和服务器程序/PID保持。见[上线记录](docs/deployment/opening-menu-0.0.59-live.md)。

2026-10-13：**App 2.0.27（22700）与Go于16:23上线，聊天0.1.10/资源0.0.58-dev于16:29发布**（PR #123，源码 `669c8f7a782c`）。附近消息五分钟过期、游戏私聊排序与App实时恢复已修复；世界内路标关闭，地图与石碑联动保留。原业务数据与游戏服PID保留，三常驻节点在线、MEK停止。维护 `.worktrees/game-chat-ldlib` / `xiwanzi/chat-expiry-order`，见[上线记录](docs/deployment/chat-recovery-2.0.27-live.md)。

2026-10-08：**App 2.0.21（22100）、Web 2.0.30 与配套 Go 已于 23:27 上线**（PR #83，源码 `b5e7cd6ef589`）。购物袋支持跨店一次确认、按店独立订单/游戏邮件/退款，优惠券页恢复原样，其他优惠展示保持。57 条账号、39 订单、5 委托、3,085 件在售商品、12 工单、2 注销记录保留；三常驻节点在线，MEK 继续停止。维护 `.worktrees/catalog-scale` / `xiwanzi/release-v221-live`，见[上线记录](docs/deployment/deuterium-2.0.21-live.md)。下方为较早记录。

2026-10-08：**App 2.0.20（22000）、Web 2.0.29 与配套 Go 已于 22:32 上线**（PR #81，源码 `cda164865864`）。商城双列卡片、图片主体适配与六大分类已发布；普通价格浅色黑/深色白，优惠价及信息低饱和粉色，按账号真实优惠预估接入。57 条账号、38 订单、5 委托、3,085 件在售商品、12 工单、2 注销记录保留；三常驻节点在线，MEK 16:45 已停止并保持。维护 `.worktrees/catalog-scale` / `xiwanzi/release-v220-live`，见[上线记录](docs/deployment/deuterium-2.0.20-live.md)。下方为较早记录。

2026-10-08：**App 2.0.19（21900）、Web 2.0.28 与配套 Go 已于 14:31 上线**（PR #79，源码 `42ea3dcd05a1`）。修复不限库存误判售罄、商城全量加载与离页红错，增加分页及图片预取；56 条账号、38 订单、5 委托、3,085 件在售商品、12 工单、2 注销记录保留，四节点在线。维护 `.worktrees/catalog-scale` / `xiwanzi/catalog-scale`，见[上线记录](docs/deployment/deuterium-2.0.19-live.md)。下方为较早记录。

2026-10-05：**App 2.0.18（21800）已于 23:53 上线**（PR #76，源码 `2e671cc98d32`）。找回密码等输入框采用实色底、常驻边框和清晰标签，浮层玻璃保留。Web 2.0.27、Go 程序及游戏插件保持；55 账号、27 订单、5 委托保留，四节点在线。维护 `.worktrees/registration-profile-check`，见[上线记录](docs/deployment/deuterium-2.0.18-live.md)。下方为较早记录。

2026-10-05：**App 2.0.17（21700）、Web 2.0.27、官网/客服中心账号脚本与配套 Go 已于 13:32 上线**（PR #74，源码 `0248496b034a`）。修复验证码额度/冷却、App 弹窗重开及晚到响应，统一四端文案；SMTP 明确无效地址冻结 24 小时，EventBridge 异步回执仍待配置。50 账号、27 订单、5 委托、12 工单、1 注销记录保留，四节点在线。维护 `.worktrees/registration-profile-check` / `xiwanzi/auth-flow-audit`，见[上线记录](docs/deployment/auth-flow-2.0.17-live.md)。下方为较早记录。

2026-10-05：**App 2.0.16 更新清单已修正**。最低适用版本从 0 改为 1，解决旧 App 的“更新版本无效”；APK、签名及其余 16 个发布条目保留。服务 active / ready，20004/21300/21500 可获取 21600，21600 无重复更新。发布时必须核对客户端解析约束，见[修复记录](docs/deployment/v216-update-manifest-fix-20261005.md)。下方为较早记录。

2026-10-05：**App 2.0.16（21600）、Web 2.0.26 与配套 Go 已上线**（PR #71，源码 `4971fd7b61fd`）。正版 ID 校验移至获取验证码，点击立即倒计时，失败恢复按钮且不发邮件；成功结果缓存 10 分钟，换邮箱复用，最后注册不再查询。官网/Help 同步更新；46 账号、27 订单、5 委托、12 工单、1 注销记录及四节点保留。维护 `.worktrees/registration-profile-check` / `xiwanzi/registration-code-check`，见[上线记录](docs/deployment/registration-code-check-live-20261005.md)。下方为较早记录。

2026-10-05：**注册正版 ID 校验配套 Go 已于 11:26 上线**（PR #68，源码 `6271f468b718`）。官网、Help、社区和 App 的最终注册提交统一查询正版玩家档案，不存在则拒绝注册，查询故障提示重试；验证码不消耗，独立游戏身份验证保留。App 2.0.15 / Web 2.0.25 保持；45 账号、27 订单、5 委托、12 工单、1 注销记录及四节点保留。维护 `.worktrees/registration-profile-check` / `xiwanzi/registration-profile-check`，见[上线记录](docs/deployment/registration-profile-check-live-20261005.md)。下方为较早记录。

2026-10-04：**邮件主题与数字字体修正已于 20:07 上线**（PR #64，源码 `cf94741842ec`）。工单号仅留正文，不再写入邮件主题；工单号恢复 Consolas，普通数字恢复 Arial，中文和排版不变。App 2.0.15 / Web 2.0.24 保持。维护 `.worktrees/mail-identity-support` / `xiwanzi/mail-subject-digits`，见[上线记录](docs/deployment/mail-subject-digits-live.md)。下方为较早记录。

2026-10-04：**Web 2.0.24 与配套 Go 已于 19:56 上线**（PR #62，源码 `0a6fea8d047c`）。工作台支持主动建单、首封带编号人工邮件且不发自动回执；超级管理员可修改其他账号游戏 ID，保留原游戏身份及资产。App 2.0.15 不变；本轮基线 40 账号、6 工单、27 订单、5 委托、1 注销记录保留，四节点在线。维护 `.worktrees/mail-identity-support` / `xiwanzi/outbound-support-ticket`，见[上线记录](docs/deployment/outbound-support-2.0.24-live.md)。下方为较早记录。

2026-10-04：**Web 2.0.23 与配套 Go 已于 14:09 上线**（PR #60，源码 `77af63a1b290`）。自动工单回执、人工回复及相关网页称呼统一为“Deuterium IX 玩家支持”；邮箱地址、样式和配置保持。App 2.0.15 不变，8 账号、4 工单、27 订单、5 委托保留。维护 `.worktrees/mail-identity-support` / `xiwanzi/player-support-name`，见[上线记录](docs/deployment/player-support-name-live.md)。下方为较早记录。

2026-10-04：**工单工作区 Web 2.0.22 与配套 Go 已于 13:46 上线**（PR #58，源码 `939d0b780f77`）。默认收起全站导航，铺满可用宽高，固定回复区，前置处理操作；草稿/重试与中文搜索修复。App 2.0.15、邮件样式和配置保持；8 账号、3 工单、27 订单、5 委托、0 注销记录保留，四节点在线。维护 `.worktrees/mail-identity-support` / `xiwanzi/support-workspace-efficiency`，见[上线记录](docs/deployment/support-workspace-2.0.22-live.md)。下方为较早记录。

2026-10-04：**邮件系统已接通，最终 Go 修正于 12:49 上线**（PR #55、#56，源码 `b0b6686cf1ad`）。DM、客服 SMTP/IMAP、内部邮箱已启用并完成真实联调；品牌图改用 HTTPS 外链，回执去掉问题摘要，按用户要求保留署名/页脚布局与 Gmail 折叠行为。App 2.0.15 / Web 2.0.21 保持；原 7 账号保留，启用后新增 1 个邮箱注册账号，当前 8 账号、27 订单、5 委托、0 注销记录。维护 `.worktrees/mail-identity-support` / `xiwanzi/mail-live`，见[上线记录](docs/deployment/mail-system-live-20261004.md)。下方候选及未接通说明均为历史状态。

2026-10-04：**App 2.0.15（21500）、社区 Web 2.0.21 与配套 Go 已于 09:30 上线**（PR #53，源码 `6d5a2fdde8f1`）。新增邮箱注册、独立游戏验证、中文昵称、邮件系统及邮件工单，旧账号保持已验证。7 账号、27 订单、5 委托和 0 注销记录保留，四个 Core 节点在线；游戏插件未更换。新 DM / 企业邮箱授权码尚待后台配置，邮箱注册及工单收发暂未启用；旧审核邮件沿用原 SMTP。维护 `.worktrees/mail-identity-support` / `xiwanzi/release-v215-live`，见[上线记录](docs/deployment/deuterium-2.0.15-live.md)。下方未部署说明为早期候选记录。

2026-10-04：**按用户确认名单，00:08 已物理清除其余 65 条 App 账号，仅保留七个 active 账号，注销记录为 0**。关联 1 笔 App 转账、2 笔取消未付款订单和 3 个所属图片对象已清理；其余 27 订单、5 委托及游戏余额/存档/白名单保持。见[清理记录](docs/deployment/account-purge-20261004.md)，不得从旧备份自动恢复。注册/找回/改密验证码模板及邮件卡片改动仅为本机候选，未上线，见[模板验收](docs/qa/verification-mail-templates-20261004.md)。


2026-10-03：**邮件系统、独立游戏验证与邮件工单本机候选已完成，未推送/部署**。工作区 `.worktrees/mail-identity-support`、分支 `xiwanzi/mail-identity-support`。邮箱注册后再补 QQ/游戏验证码；旧账号默认已验证，中文昵称与游戏 ID 分离。工单按客服邮箱收件生成日期＋四位随机编号，DM 自动回执，企业邮箱人工回复；没有玩家在线工单对话。原管理员为超级管理员，新管理员仅工单。真实 SMTP/IMAP 授权码及真机验收待完成，App/Web 发布版本未变。见[本机验收](docs/qa/mail-identity-support.md)。下方为既有发布记录。


2026-09-28：**Web 2.0.20 与配套 Go 已上线**（PR #50、#51）。后台新增「模型管理」，统一栏目、界面、Deuterium ID 登录和管理员权限，复用原 Velocity API；256 模型、476 文件、模型 revision 11 / 配置 revision 3 与四节点 READY 保留。App 仍为 2.0.14；72 条账号、29 笔订单、5 个委托及已有注销记录保留，游戏插件未更换。维护 `.worktrees/chat-domain` / `xiwanzi/model-management-record`，见[上线记录](docs/deployment/model-management-live.md)。下方为较早状态。

2026-09-26：**App 2.0.14（21400）与配套 Go 已于 20:00 上线**（PR #48，源码 `4e95ae0df69b`）。公共聊天在线名单改读 Core 实时快照，页面返回回调在键盘收起期间保持有效。Web 2.0.19、游戏插件和配置保留；72 条账号记录、29 笔订单、5 个委托及已有注销记录保留。维护 `.worktrees/chat-domain` / `xiwanzi/release-v214-live`，见[部署记录](docs/deployment/deuterium-2.0.14-live.md)。

2026-09-19：**玩家社区域名 https://chat.deuteriumix.com 已于 21:12 上线**。独立 HTTPS 证书和自动续期验证通过，Go 精确兼容新旧来源，App 2.0.13 / Web 2.0.19 保持；旧 IP 接口和启动器入口保留。源码 `a1064819ac0a`，维护 `.worktrees/chat-domain`，见[上线记录](docs/deployment/chat-domain-live.md)。下方为较早记录。

2026-09-15：**Go 并发优化已于 20:47:45 上线**（PR #43，源码 `a824088cb518`）。请求分组限额、公共聊天读取合并、AI/密码短时等待与管理员容量接口已部署。App 2.0.13、Web 2.0.19、启动器 0.6.1、申请站 1.0.2 保持；配置、71 条账号记录、27 笔订单和 5 个委托保留。按用户要求未运行测试或压测。维护工作区 `C:/DeiteriumAPP-IX/.worktrees/backend-concurrency`，见[上线记录](docs/deployment/backend-concurrency-live.md)。下方为较早记录。

2026-09-13：**申请站 1.0.2 配色优化已于 20:48 上线**（PR #41）。主页面减少装饰性亮黄，弹窗查询改深色按钮、返回改描边按钮；仅替换静态 CSS，申请、审核、邮件和其他组件不变。见[上线记录](docs/deployment/admission-1.0.2-live.md)。下方为较早记录。

2026-09-13：**Web 2.0.19、配套 Go 与启动器 0.6.1 已上线**（PR #39）。管理页新增程序签名发布，继续使用现有 OSS；启动器内嵌玩家社区并接入独立更新器，实际自动升级通过。App 保持 2.0.13，业务与白名单状态保留。见[上线记录](docs/deployment/dlauncher-0.6.1-live.md)。下方为较早记录。

2026-09-13：**审核邮件配色保护与小尺寸徽章已于 10:47 上线**（PR #37）。通过/拒绝模板均采用已确认的锁色处理，内嵌图标从 1.82 MB 降至 32.5 KB，仍使用 CID 附件。SMTP 配置及审批流程保留，Web 仍为 2.0.18、App 2.0.13。见[上线记录](docs/deployment/mail-appearance-live.md)。下方为较早记录。

2026-09-13：**Web 2.0.18、配套 Go 与白名单审核邮件已于 02:05 上线**（PR #35）。通过/拒绝结果自动发至申请人的 QQ 邮箱，使用后台已保存的 SMTP 配置；两种模板实际测试均被接收。支持模板预览、测试、投递状态与失败重试，过期通知取消。App 仍为 2.0.13，70 条已撤销资格未恢复。见[上线记录](docs/deployment/admission-review-email-live.md)。下方为较早记录。

2026-09-13：**SMTP 队列修复已于 00:36 上线**，解决 MariaDB 10.5 不支持 `SKIP LOCKED` 导致测试邮件始终待发送的问题。两封已排队测试邮件已被 QQ SMTP 接收，配置/授权码保留。Web 仍为 2.0.17，App 2.0.13，70 条已撤销白名单资格未恢复；审批通过邮件尚待正文确定。见[上线记录](docs/deployment/smtp-queue-fix-live.md)。下方为较早记录。

2026-09-13：**Web 2.0.17 已于 00:10 上线**，修复白名单列表切回申请记录导致全屏白屏。按用户最新要求，00:14 已撤销全部 70 个现有白名单资格，当时有效资格为 0；账号及历史保留，在线连接未主动断开。以后须重新申请审批，禁止自动恢复旧导入资格。App、Go、Gateway 沿用原版。见[部署与重置记录](docs/deployment/web-2.0.17-live.md)。下方为较早记录。

2026-09-12：独立通行申请站、Web 2.0.16、Go 准入接口及 Velocity Gateway 1.0.0 已上线，App 保持 2.0.13。代理在正版认证后统一核对 UUID 白名单，再选择上次子服；首次默认 Amiya。新玩家申请不依赖 App 注册，管理端审核及手动管理，QQ群成员由管理员核对。申请、当前资格和操作历史独立保存，移除资格不删除玩家或交易记录。部署与当前入口见[通行申请上线记录](docs/deployment/admission-v1-live.md)。以下为较早上下文。

2026-09-12：**App 2.0.13（21300）、Web 2.0.15 与配套 Go 已于 19:15 上线**，源码 `15690c138288`（PR #26），xiwanzi 更新提醒已推送。工作台支持当前管理员密码确认的永久注销，App/Web 同步清理关联展示，历史交易匿名保留；已部署启动器 0.5 源码一并合入，内容版本 5 和配置未改。71 个账号、27 笔订单、5 个委托保留，四节点在线；本轮未注销真实账号。维护工作区 `.worktrees/account-erasure`，分支 `xiwanzi/release-v213-live`。见[部署记录](docs/deployment/deuterium-2.0.13-live.md)。下方为较早记录。

2026-09-12：**管理员永久注销账号候选已完成本机实现与验收，尚未推送/部署**。源码 `038718b068fb`，工作区 `.worktrees/account-erasure`、分支 `xiwanzi/account-erasure`。工作台校验当前管理员密码，后端清理账号与展示数据，App/Web 同步清理缓存，历史交易匿名保留。没有操作真实旧号；线上仍为 2.0.12。见[验收记录](docs/qa/account-permanent-deletion.md)。

2026-09-12：**DLauncher 0.5.0 / Web 2.0.14 配套 Go 已上线**，切换为整包发布后仅 McPatch 更新，删除首次安装接口和配置；Deuterium IX 使用 D9 徽章及 DSA 素材，其他入口隐藏。本轮从 origin/main 4a9c249 继续，保留并上线同步超时终态修复。维护 `.worktrees/dlauncher-bundled` / `feat/dlauncher-bundled`；用户要求本轮不生成完整游戏包。见[部署记录](docs/deployment/dlauncher-0.5-live.md)。

2026-09-12 维护收口：启动器管理按本轮授权通过 PR 归入 main，后续 Web/Go 迭代从最新 origin/main 开分支；`.worktrees/dlauncher-admin` 保留为部署追溯入口。首次上线源码为 `fdf788c`。本轮 review 修正同步超时后无法保存终态的问题，未重跑测试、未重新部署；该修正需随下一次获授权的 Go 发布上线。下方“代码仅本地提交”为首次部署时的历史状态。

2026-09-12：**Web 2.0.13 与启动器配套 Go 已上线**，管理导航新增“启动器”，连接同机 McPatch 原生 Web 与 OSS 同步，初始基线已发布。Android 沿用 2.0.12；71 个账号、27 笔订单、5 个委托保留，游戏服务未重启。本轮维护 `.worktrees/dlauncher-admin` / `feat/dlauncher-admin`，代码仅本地提交。见[部署与验证](docs/deployment/dlauncher-0.4-live.md)。下方为较早记录。

2026-09-11：**App 2.0.12（21200）与配套 Go 已于 22:18 上线**，源码 `a0ed4446d074`（PR #23），xiwanzi 更新提醒已推送。注册验证码不再依赖密码，忘记密码浮层接入统一背景模糊；Web 沿用 2.0.10，游戏插件未更换。71 个账号、27 笔订单和 5 个委托全部保留，四个游戏节点当前在线，旧号未注销。维护工作区 `.worktrees/auth-registration-glass`，分支 `xiwanzi/release-v212-live`。见[部署记录](docs/deployment/deuterium-2.0.12-live.md)。下方为较早记录。

2026-09-11：**App 2.0.11（21100）已发布**，合并源码 `46d4c5e76eb5`，更新说明“大幅提升流畅度，提升系统稳定性。”；账号提醒已推送。Web/Go 沿用 2.0.10，游戏组件沿用既有部署。当前维护工作区 `.worktrees/performance-audit-v210`、分支 `xiwanzi/release-v211-live`，见[部署记录](docs/deployment/deuterium-2.0.11-live.md)。

2026-09-11：全局过度绘制专项已在连接真机上完成覆盖验收，维护分支 `xiwanzi/global-overdraw-v210`，工作区 `.worktrees/performance-audit-v210`。保留玻璃原采样，只缓存静态屏幕背景；App 版本仍为 2.0.10（21000），未线上发布。见[真机结果与边界](docs/qa/app-global-overdraw-v210.md)。

2026-09-11：本机新增 App 性能优化候选，工作区 `.worktrees/performance-audit-v210`、分支 `xiwanzi/performance-equivalence-v210`；保持原视觉和动画参数，版本仍为 2.0.10（21000），未发布。见[同效验收与限制](docs/qa/app-performance-equivalence-v210.md)。下方为线上版本记录。

2026-09-11：**App 2.0.10（21000）、Web 2.0.10 与配套 Go 已于 02:38 上线**，源码 `65c08a290896`（PR #18、#19），已向 xiwanzi 推送更新提醒。优惠券统一应用内通知、停用券/商品可删除、全额退款返券及启动绘制优化已生效；迁移释放 1 条历史退款券占用，26 笔订单与 4 个委托保留。三常驻服在线，MEK 继续停止；游戏插件未更换。维护工作区 `.worktrees/commerce-refunds-performance`，分支 `xiwanzi/release-v210-live`。见[部署记录](docs/deployment/deuterium-2.0.10-live.md)。下方为较早记录。

2026-09-11：优惠券生命周期、商品删除及启动绘制修正已通过 [PR #18](https://github.com/xiwanzi/DeiteriumAPP-IX/pull/18) 合入 main，源码 `303db4ee57f9`。App/Web 2.0.10（App 21000）准备测试交付；线上部署状态见[2.0.10 交付记录](docs/deployment/deuterium-2.0.10-handoff.md)。下方“尚未推送、合并”是开发阶段记录。

2026-09-11：本机已完成优惠券应用内通知、停用/草稿券删除、商店商品删除、全额退款返券与启动/加载绘制优化。维护工作区 `.worktrees/commerce-refunds-performance`，分支 `xiwanzi/commerce-refunds-performance`；[验收记录](docs/qa/commerce-refunds-performance.md)。Go 全量竞态/集成、Android 134、Web 65 及原生通知检查通过；尚未推送、合并或部署，线上仍为下述 2.0.9。

2026-09-10：**App 2.0.9（20901）、Web 2.0.9、配套 Go/Core 1.0.2、Mail 0.6.2/API3、Bridge 0.6.1 与 XConomy .3 已于 16:36 上线**。主源码 `e4e4c4b7791c`（PR #16），Mail `67db93749bf9`（PR #2）。支持优惠券草稿与批量发放、同批提醒去重、折扣/周期限购及组合交付；22 笔订单、4 个委托和原配置保留，三常驻服恢复，MEK 继续停止禁领。维护工作区 `.worktrees/commerce-v209`，分支 `xiwanzi/release-v209-live`。见[部署记录](docs/deployment/deuterium-2.0.9-live.md)。下方为较早记录。

2026-09-10：**App 2.0.8（20800）、Web 2.0.8 与配套 Go 已于 11:13 上线**，源码 `6566bb617f8d`（PR #14）。默认弧线 D 与双子节图标已内置，管理员可远程切换；线上默认无横幅。22 笔订单、4 个委托及原配置保留。本轮未重跑测试。当前维护工作区 `.worktrees/launcher-icons-v208`，分支 `release/v208-live`。下方为较早记录。 见 [2.0.8 部署记录](docs/deployment/deuterium-2.0.8-live.md)。

2026-09-10：**App 2.0.7（20700）、Web 2.0.7 与配套 Go 已于 03:22 上线**，源码 `4262edd84c03`（PR #12）。Saki 支持逐日递减、最低 5 天差价的升级，严格保留原到期时间；购买确认展示升级实付与淡色删除线原价，不显示公式。空会话才显示输入框上方套餐入口。用户已开启的套餐配置及 11 笔订单、4 个委托保留，三常驻服在线，MEK 离线。当前维护工作区 `.worktrees/saki-upgrade-checkout`，分支 `release/v207-live`。下方为较早发布记录。 见 [2.0.7 部署记录](docs/deployment/deuterium-2.0.7-live.md)。

2026-09-09：**App 2.0.6（20600）、Web 2.0.6 与配套 Go 已上线**。App/Web 源码 `4d9a001ebd0e`（PR #9），Go 最终源码 `515c87fd4936`（PR #10）。账单、Saki/账号后台、头像裁切、购买订单与通知优化见 [2.0.6 部署记录](docs/deployment/deuterium-2.0.6-live.md)。Saki 售卖仍关闭；本轮工作区 `.worktrees/saki-admin-experience`，分支 `release/v206-live`。下方 2.0.5 及更早状态为历史。

2026-09-09，当前 App 2.0.5（20500），见 [2.0.5](docs/deployment/deuterium-2.0.5-live.md)；Web 2.0.5 与配套 Go（`0386f1bd5664`）已部署，见 [网页 2.0.5](docs/deployment/web-2.0.5-live.md)。Core 1.0.1 与 XConomy .2 沿用 [2.0.4](docs/deployment/deuterium-2.0.4-live.md) 配套版本。原生 Android、同源网站与统一后端连接真实账号、信用点、聊天、商城、市场及委托；游戏内收支与 App 交易使用同一份已提交账本。

## 架构

```text
Android (Kotlin / Compose)    Web (React / Vite 静态站)
             \                 /
              HTTPS / API v1、WebSocket、SSE
                        |
                  Go 后端 + MariaDB
                        |
                   Core 主动 WSS 连接
                        |
               Deuterium Core (Java 21 / Youer 1.21.1)
                 |          |          |
          XConomy资金API  独立Mail API3  Sync保存屏障
```

`backend-next` 的 Go 模块路径仍为旧仓库名，用于保持现有导入和构建兼容；Git 远程位置已经独立，不据此导入旧 Ktor 实现。网站生产只发布静态 `dist`，Node 网关不是生产前置依赖。

## 领域与边界

| 术语 | 含义 |
| --- | --- |
| Deuterium ID | 第一方账号体系；游戏服务解析的 UUID 是资产身份权威，QQ/游戏名是登录或展示标识 |
| 信用点 | XConomy 自有数据库和受控事务为权威；后端保存业务状态，不直接改游戏经济表 |
| DIMA / DaoYu | 官方收入 / 预付托管专户；不是可登录的普通玩家账号 |
| 物品版本 | Core 保存不可变 ItemStack 快照，商品与邮件锁定版本及摘要 |
| 邮箱 | 独立 Mail 是发放与持久权益权威；Bridge/UI/Core 通过公开 API 调用 |
| 保存屏障 | Sync 确认玩家完成加载、独占会话及库存事务提交，回执未知时隔离并只读恢复 |
| 市场 / 委托 | 由后端状态机处理担保、退款、验收、到期和平台介入，不由客户端自行结算 |
| 付款动画 | 1–3 秒模拟识别期间启动已确认请求；识别结束且后端成功才显示勾，不采集人脸 |
| 图片生命周期 | 活跃业务引用阻止回收；无当前用途后迁入专用 gc 前缀，30 天对象生命周期，订单记录保留 |

## 当前节点

Login 禁领且不参加库存同步；Amiya/Odyssey 使用 Sync 共享库存域 `survival`；MEK 同域，但保持停止及禁领，模组兼容未完成验收。四服安装同批 Core/Mail/Sync/XConomy，经济权威节点只能配置一个。

## 产品结构

四入口为商城、市场、信息、我的。钱包/账单/转账在我的；公共聊天、私聊、AI、公告和委托在信息。主题只有系统/浅色/深色；保留柔光玻璃、灵动视效和动态追光。完整已确认规则见 [产品基线](docs/prd/deuterium-current.md)。

旧工作区路径、旧 Ktor/Mohist 1.20.1 和 01–09 演示迭代保留为历史追溯资料，不是当前部署指令。整理前入口快照见 [历史入口](docs/archive/pre-consolidation/CONTEXT.md)。
