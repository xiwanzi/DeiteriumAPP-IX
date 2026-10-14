2026-10-15：**启动器 0.6.10 于03:00发布，Web 2.0.41 与配套 Go 于02:58上线，03:03公网核验通过**（DeLauncher PR #11 / 主 PR #132）。共享更新运行库并支持签名文件级增量，更新器73.80→2.72 MiB，完整包271.75→201.10 MiB；更新器样式保持。首次0.6.9迁移仍需完整包。原账号、业务、配置及游戏资源0.0.60-dev保留，App2.0.29保持，三常驻节点在线、MEK停止，本轮未重启游戏服。维护 `.worktrees/launcher-incremental` / `xiwanzi/launcher-incremental`。见[上线记录](deployment/launcher-incremental-0.6.10-live.md)。下方为历史状态。

2026-10-14：**App 2.0.29（22900）、Web 2.0.40 与配套 Go 已于14:19上线，14:21公网核验通过**（PR #130，源码 `a8377b1387d1`）。上线官方资料查询与登录容量优化、App 长期/Web 30 天会话、设备管理和未验证身份占用处理；未验证账号不自动清理，已验证身份不换绑。61 条账号记录、47 订单、5 委托、3,085 在售商品、12 工单、2 注销记录、8 个已验证身份及原配置保留，105 张原表指纹核对一致。三常驻节点在线、MEK停止；启动器0.6.8/资源0.0.60-dev保持，本轮未重启游戏服。维护 `.worktrees/account-experience-capacity` / `xiwanzi/account-experience-capacity`，见[上线记录](deployment/account-experience-2.0.29-live.md)。下方候选说明为历史状态。

2026-10-13：**App 2.0.27（22700）与 Go 于16:23上线，聊天0.1.10/资源0.0.58-dev于16:29发布**（PR #123）。修复附近五分钟过期、游戏私聊排序和 App 实时恢复；关闭世界内路标显示，保留地图与石碑联动。原业务记录、配置和游戏服PID保留，三常驻节点在线、MEK停止。维护 `.worktrees/game-chat-ldlib` / `xiwanzi/chat-expiry-order`；见[上线记录](deployment/chat-recovery-2.0.27-live.md)。

2026-10-14：**账号体验与容量优化本机候选已完成，未发布。** 维护 `.worktrees/account-experience-capacity` / `xiwanzi/account-experience-capacity`。统一正版查询排队与合并，登录来源/失败计数分离，App 长期/Web 30 天会话，设备管理和未验证身份冲突闭环；本机百人登录与通行隔离、账号回归、Web 和独立 Android QA 通过。见[验收](qa/account-experience-capacity-20261014.md)、[决策](adr/0011-account-sessions-and-profile-conflicts.md)及[契约/回退](contracts/account-experience-capacity.md)。生产仍以已有上线记录为准。

2026-10-13：**资源0.0.59-dev于17:03发布，17:04公网核验通过**（DeLauncher PR #6）。修复进服设置页关闭后重新黑屏渐出的遗漏流程，正常首次过渡保留；仅更新主开屏JAR及显示版本。App2.0.27、聊天0.1.10、路标配置和服务器程序/PID保持。见[上线记录](deployment/opening-menu-0.0.59-live.md)。

2026-10-12：**App 2.0.26（22600）私聊表情与媒体已于 23:50 上线**（PR #119，源码 `01cdac40bbc4`）。私聊与公屏共用 Emoji、图片/GIF、个人贴纸及预览/收藏/重试；队列按会话隔离，重启恢复保持原收件人。61 账号、47 订单、5 委托、3,085 件在售商品、12 工单、2 注销记录保留，三常驻节点在线、MEK 停止。Web 2.0.38、主 Go 程序、图片审核及游戏插件保持。维护 `.worktrees/chat-media` / `xiwanzi/release-v226-live`，见[上线记录](deployment/deuterium-2.0.26-live.md)。

2026-10-12：**游戏客户端 0.0.50-dev 已于 00:31 完成上线核验**（PR #105、#106）。中性黑色渐变与清晰 Logo，分享改为单次当前位置和世界频道当前主手物品；修复物品翻译键及重复消息，保留富文本交互。此前 48 的小屏、按键、右上小地图和音乐修复保持。Go/Core/App/Web 及业务数据不变，三常驻节点在线、MEK 停服。见[上线记录](deployment/game-chat-0.0.50-live.md)。

2026-10-12：**头衔与铭牌系统、聊天修复已于04:36发布，04:39完成公网验收**。游戏资源0.0.51-dev、聊天0.1.7、MMO0.1.8、Core1.0.5、Web2.0.36与配套Go上线（主PR #108、MMO PR #1）。头衔介绍/公开属性、LP授权与到期回退、原生铭牌及接收提示音已接通；61账号、47订单、5委托、3,085在售商品、12工单、2注销记录及原配置保留。三常驻节点在线，MEK保持停止，App仍2.0.25。见[上线记录](deployment/nameplates-0.0.51-live.md)。

2026-10-11：**游戏客户端 0.0.47-dev 与配套 Go 已于 23:27 完成上线核验**（PR #101、#102）。服务器持久缓存离线头像，TAC 配置键解绑、Alt+T 保留表情轮盘，完整聊天获得 IMBlocker 中文输入豁免。原业务数据保留，三常驻节点在线、MEK 停服；Core 1.0.4、App 2.0.25、Web 2.0.35 保持。见[上线记录](deployment/game-chat-0.0.47-live.md)。

2026-10-11：**游戏客户端 0.0.45-dev 已于 22:55 上线**（PR #99，源码 `1207c68`）。T 快捷聊天、Alt+T 表情轮盘、Enter 完整界面；头像补齐皮肤外层。Nginx 游戏聊天 HTTP 路由已修复，xiwanzi 已自动建立会话，无需重新绑定账号。Core/Go/App/Web 与业务数据保持，三常驻节点在线、MEK 停服。见[上线记录](deployment/game-chat-0.0.45-live.md)。

2026-10-11：**游戏客户端 0.0.44-dev、Core 1.0.4 与配套 Go 已于 22:33 完成上线核验**（PR #96、#97）。新增 LDLib2 原生聊天、图片/贴纸、私聊、附近与团队路由及专用会话权限；保留 YSM 与家具同步修复。61 账号、47 订单、5 委托、3,085 件在售商品、12 工单、2 注销记录保留，三常驻节点在线、MEK 文件升级后仍停服。App 2.0.25 / Web 2.0.35 保持。维护 `.worktrees/game-chat-ldlib`，见[上线记录](deployment/game-chat-0.0.44-live.md)。下方候选及旧发布为历史状态。

## 2026-10-11 游戏聊天本机候选（历史：后续已发布）

独立工作区 `C:/DeiteriumAPP-IX/.worktrees/game-chat-ldlib` 新增 NeoForge 1.21.1 / LDLib2 聊天客户端及配套 Core/Go：完整聊天、HUD、附近50米、现有团队路由、私聊、S3图片/贴纸。按 Web V12 与用户后续要求修正尺寸、材质、字体、抗锯齿及音效。未推送/部署，不改变下方生产版本记录。

入口：[模块说明](../game-chat-client/README.md)、[实施边界](plans/game-chat-ldlib-20261011.md)、[本机验收](qa/game-chat-ldlib-20261011.md)。交付包在本工作区 `output/deuterium-game-chat-candidate-20261011.zip`。

2026-10-10：**App 2.0.25（22500）、Web 2.0.34 已于 01:55 上线**（PR #91，源码 `984ceee1163dd`）。输入栏与表情贴纸面板独立适配浅深色、提高可读性，App 支持分别调节材质；顶部圆角与贴底布局保留。58 账号、44 订单、5 委托、3,085 件在售商品、12 工单、2 注销记录保留，三常驻节点在线、MEK 停止。维护 `.worktrees/chat-media` / `xiwanzi/release-v225-live`，见[上线记录](deployment/deuterium-2.0.25-live.md)。下方候选及旧发布为历史状态。

2026-10-10：**聊天玻璃独立参数调校已完成本机候选，未发布**。输入栏/表情面板分别保存浅深色材质，并可在“材质细节”独立调整、还原；提高可读性、降低反光，保留顶部圆角与贴底布局。维护 `.worktrees/chat-media` / `xiwanzi/chat-glass-tuning`，见[验收与默认值](qa/chat-glass-materials-20261010.md)。线上仍为下方 2.0.24 / Web 2.0.33。

2026-10-10：**App 2.0.24（22400）、Web 2.0.33 已于 00:47 上线**（PR #89，源码 `ab3893a935bd`）。聊天玻璃取样实际消息，表情面板仅顶部圆角、左右贴边并铺满底部手势区。58 账号、44 订单、5 委托、3,085 件在售商品、12 工单、2 注销记录保留；Go/数据库/游戏插件未换。见[上线记录](deployment/deuterium-2.0.24-live.md)。用户随后要求继续独立调校两组玻璃参数，后续候选须与此线上版本区分。

2026-10-09：**App 2.0.23（22300）、Web 2.0.32 已于 23:40 上线**（PR #87，源码 `e21b38b45e8a`）。按参考录屏调整输入栏、键盘/表情联动、贴纸面板和图片预览抽屉，贴纸统一自动圆角。Go/数据库/游戏插件保持；58 账号、44 订单、5 委托、3,085 件在售商品、12 工单、2 注销记录保留，三常驻节点在线、MEK 停止。维护 `.worktrees/chat-media` / `xiwanzi/release-v223-live`，见[上线记录](deployment/deuterium-2.0.23-live.md)。下方候选及旧发布均为历史。

2026-10-09：**聊天输入区动效候选已完成本机调整，尚未上线**。按用户录屏调整胶囊输入栏、键盘/表情联动、分类与紧凑贴纸面板、可拖拽图片预览，并增加静态/动态贴纸自动圆角。维护 `.worktrees/chat-media` / `xiwanzi/chat-composer-motion`，见[验收](qa/chat-composer-motion-20261009.md)。线上仍为下方 2.0.22 / Web 2.0.31。

2026-10-09：**App 2.0.22（22200）、Web 2.0.31 与配套 Go 已于 19:21 上线**（PR #85，源码 `60b0cb525842`）。公共聊天新增 Emoji/图片/GIF/10 个云端贴纸，三天媒体与去重、500MiB 本机聊天缓存、30 天历史补拉；036 迁移和真实 OSS 联调通过。58 账号、44 订单、5 委托、3,085 件在售商品、12 工单、2 注销记录保留；三常驻节点在线，MEK 继续停止。维护 `.worktrees/chat-media` / `xiwanzi/release-v222-live`，见[上线记录](deployment/deuterium-2.0.22-live.md)。下方候选与较早发布均为历史。

2026-10-09：公共聊天媒体已在 `.worktrees/chat-media` / `xiwanzi/chat-media` 完成本机候选；包含 Emoji、单图压缩、GIF、10 个个人贴纸、三天云端媒体、500MiB 本机聊天缓存与 30 天远程历史。**未推送或部署**，见[验收](qa/chat-media-20261009.md)与[契约](contracts/public-chat-media.md)。

2026-10-08：**App 2.0.21（22100）、Web 2.0.30 与配套 Go 已于 23:27 上线**（PR #83，源码 `b5e7cd6ef589`）。购物袋支持跨店一次确认、按店独立订单/游戏邮件/退款，优惠券页恢复原样，其他优惠展示保持。57 条账号、39 订单、5 委托、3,085 件在售商品、12 工单、2 注销记录保留；三常驻节点在线，MEK 继续停止。维护 `.worktrees/catalog-scale` / `xiwanzi/release-v221-live`，见[上线记录](deployment/deuterium-2.0.21-live.md)。下方为较早记录。

2026-10-08：**App 2.0.20（22000）、Web 2.0.29 与配套 Go 已于 22:32 上线**（PR #81，源码 `cda164865864`）。商城双列卡片、图片主体适配与六大分类已发布；普通价格浅色黑/深色白，优惠价及信息低饱和粉色，按账号真实优惠预估接入。57 条账号、38 订单、5 委托、3,085 件在售商品、12 工单、2 注销记录保留；三常驻节点在线，MEK 16:45 已停止并保持。维护 `.worktrees/catalog-scale` / `xiwanzi/release-v220-live`，见[上线记录](deployment/deuterium-2.0.20-live.md)。下方为较早记录。

2026-10-08：**App 2.0.19（21900）、Web 2.0.28 与配套 Go 已于 14:31 上线**（PR #79，源码 `42ea3dcd05a1`）。修复不限库存误判售罄、商城全量加载与离页红错，增加分页及图片预取；56 条账号、38 订单、5 委托、3,085 件在售商品、12 工单、2 注销记录保留，四节点在线。维护 `.worktrees/catalog-scale` / `xiwanzi/catalog-scale`，见[上线记录](deployment/deuterium-2.0.19-live.md)。下方为较早记录。

2026-10-05：**App 2.0.17（21700）、Web 2.0.27、官网/客服中心账号脚本与配套 Go 已于 13:32 上线**（PR #74，源码 `0248496b034a`）。修复验证码额度/冷却、App 弹窗重开及晚到响应，统一四端文案；SMTP 明确无效地址冻结 24 小时，EventBridge 异步回执仍待配置。50 账号、27 订单、5 委托、12 工单、1 注销记录保留，四节点在线。维护 `.worktrees/registration-profile-check` / `xiwanzi/auth-flow-audit`，见[上线记录](deployment/auth-flow-2.0.17-live.md)。下方为较早记录。

2026-10-05：**App 2.0.18（21800）已于 23:53 上线**（PR #76，源码 `2e671cc98d32`）。找回密码等输入框采用实色底、常驻边框和清晰标签，浮层玻璃保留。Web 2.0.27、Go 程序及游戏插件保持；55 账号、27 订单、5 委托保留，四节点在线。维护 `.worktrees/registration-profile-check`，见[上线记录](deployment/deuterium-2.0.18-live.md)。下方为较早记录。

2026-10-05：**App 2.0.16 更新清单已修正**。最低适用版本从 0 改为 1，解决旧 App 的“更新版本无效”；APK、签名及其余 16 个发布条目保留。服务 active / ready，20004/21300/21500 可获取 21600，21600 无重复更新。发布时必须核对客户端解析约束，见[修复记录](deployment/v216-update-manifest-fix-20261005.md)。下方为较早记录。

2026-10-05：**App 2.0.16（21600）、Web 2.0.26 与配套 Go 已上线**（PR #71，源码 `4971fd7b61fd`）。正版 ID 校验移至获取验证码，点击立即倒计时，失败恢复按钮且不发邮件；成功结果缓存 10 分钟，换邮箱复用，最后注册不再查询。官网/Help 同步更新；46 账号、27 订单、5 委托、12 工单、1 注销记录及四节点保留。维护 `.worktrees/registration-profile-check` / `xiwanzi/registration-code-check`，见[上线记录](deployment/registration-code-check-live-20261005.md)。下方为较早记录。

2026-10-05：**注册正版 ID 校验配套 Go 已于 11:26 上线**（PR #68，源码 `6271f468b718`）。官网、Help、社区和 App 的最终注册提交统一查询正版玩家档案，不存在则拒绝注册，查询故障提示重试；验证码不消耗，独立游戏身份验证保留。App 2.0.15 / Web 2.0.25 保持；45 账号、27 订单、5 委托、12 工单、1 注销记录及四节点保留。维护 `.worktrees/registration-profile-check` / `xiwanzi/registration-profile-check`，见[上线记录](deployment/registration-profile-check-live-20261005.md)。下方为较早记录。

2026-10-04：**工单归并修复 Web 2.0.25 与配套 Go 已于 20:52 上线**（PR #66，源码 `54c389cc9a43`）。无引用头回复通过正文编号与同邮箱接续原工单；3 组误拆记录已合并，完整历史保留，已关闭默认移出全部但可搜索，待处理优先。App 2.0.15 保持；41 账号、27 订单、5 委托、1 条已有注销记录保留，工单 15→12。维护 `.worktrees/mail-identity-support` / `xiwanzi/support-threading-fix`，见[上线记录](docs/deployment/support-threading-2.0.25-live.md)。下方为较早记录。

# 当前项目状态

2026-10-04：**邮件主题与数字字体修正已于 20:07 上线**（PR #64，源码 `cf94741842ec`）。工单号仅留正文，不再写入邮件主题；工单号恢复 Consolas，普通数字恢复 Arial，中文和排版不变。App 2.0.15 / Web 2.0.24 保持。维护 `.worktrees/mail-identity-support` / `xiwanzi/mail-subject-digits`，见[上线记录](deployment/mail-subject-digits-live.md)。下方为较早记录。

2026-10-04：**Web 2.0.24 与配套 Go 已于 19:56 上线**（PR #62，源码 `0a6fea8d047c`）。工作台支持主动建单、首封带编号人工邮件且不发自动回执；超级管理员可修改其他账号游戏 ID，保留原游戏身份及资产。App 2.0.15 不变；本轮基线 40 账号、6 工单、27 订单、5 委托、1 注销记录保留，四节点在线。维护 `.worktrees/mail-identity-support` / `xiwanzi/outbound-support-ticket`，见[上线记录](deployment/outbound-support-2.0.24-live.md)。下方为较早记录。

2026-10-04：**Web 2.0.23 与配套 Go 已于 14:09 上线**（PR #60，源码 `77af63a1b290`）。自动工单回执、人工回复及相关网页称呼统一为“Deuterium IX 玩家支持”；邮箱地址、样式和配置保持。App 2.0.15 不变，8 账号、4 工单、27 订单、5 委托保留。维护 `.worktrees/mail-identity-support` / `xiwanzi/player-support-name`，见[上线记录](deployment/player-support-name-live.md)。下方为较早记录。

2026-10-04：**工单工作区 Web 2.0.22 与配套 Go 已于 13:46 上线**（PR #58，源码 `939d0b780f77`）。默认收起全站导航，铺满可用宽高，固定回复区，前置处理操作；草稿/重试与中文搜索修复。App 2.0.15、邮件样式和配置保持；8 账号、3 工单、27 订单、5 委托、0 注销记录保留，四节点在线。维护 `.worktrees/mail-identity-support` / `xiwanzi/support-workspace-efficiency`，见[上线记录](deployment/support-workspace-2.0.22-live.md)。下方为较早记录。

2026-10-04：**邮件系统已接通，最终 Go 修正于 12:49 上线**（PR #55、#56，源码 `b0b6686cf1ad`）。DM、客服 SMTP/IMAP、内部邮箱已启用并完成真实联调；品牌图改用 HTTPS 外链，回执去掉问题摘要，按用户要求保留署名/页脚布局与 Gmail 折叠行为。App 2.0.15 / Web 2.0.21 保持；原 7 账号保留，启用后新增 1 个邮箱注册账号，当前 8 账号、27 订单、5 委托、0 注销记录。维护 `.worktrees/mail-identity-support` / `xiwanzi/mail-live`，见[上线记录](deployment/mail-system-live-20261004.md)。下方候选及未接通说明均为历史状态。

2026-10-04：**App 2.0.15（21500）、社区 Web 2.0.21 与配套 Go 已于 09:30 上线**（PR #53，源码 `6d5a2fdde8f1`）。新增邮箱注册、独立游戏验证、中文昵称、邮件系统及邮件工单，旧账号保持已验证。7 账号、27 订单、5 委托和 0 注销记录保留，四个 Core 节点在线；游戏插件未更换。新 DM / 企业邮箱授权码尚待后台配置，邮箱注册及工单收发暂未启用；旧审核邮件沿用原 SMTP。维护 `.worktrees/mail-identity-support` / `xiwanzi/release-v215-live`，见[上线记录](deployment/deuterium-2.0.15-live.md)。下方未部署说明为早期候选记录。

2026-10-04：**按用户确认名单，00:08 已物理清除其余 65 条 App 账号，仅保留七个 active 账号，注销记录为 0**。关联 1 笔 App 转账、2 笔取消未付款订单和 3 个所属图片对象已清理；其余 27 订单、5 委托及游戏余额/存档/白名单保持。见[清理记录](deployment/account-purge-20261004.md)，不得从旧备份自动恢复。注册/找回/改密验证码模板及邮件卡片改动仅为本机候选，未上线，见[模板验收](qa/verification-mail-templates-20261004.md)。


2026-10-03：**邮件系统、独立游戏验证与邮件工单本机候选已完成，未推送/部署**。工作区 `.worktrees/mail-identity-support`、分支 `xiwanzi/mail-identity-support`。邮箱注册后再补 QQ/游戏验证码；旧账号默认已验证，中文昵称与游戏 ID 分离。工单按客服邮箱收件生成日期＋四位随机编号，DM 自动回执，企业邮箱人工回复；没有玩家在线工单对话。原管理员为超级管理员，新管理员仅工单。真实 SMTP/IMAP 授权码及真机验收待完成，App/Web 发布版本未变。见[本机验收](qa/mail-identity-support.md)。下方为既有发布记录。


2026-09-28：**Web 2.0.20 与配套 Go 已上线**（PR #50、#51）。后台新增「模型管理」，统一栏目、界面、Deuterium ID 登录和管理员权限，复用原 Velocity API；256 模型、476 文件、模型 revision 11 / 配置 revision 3 与四节点 READY 保留。App 仍为 2.0.14；72 条账号、29 笔订单、5 个委托及已有注销记录保留，游戏插件未更换。维护 `.worktrees/chat-domain` / `xiwanzi/model-management-record`，见[上线记录](deployment/model-management-live.md)。下方为较早状态。

2026-09-28：**YSM 管理整合已完成本机实现和验证，尚未推送/合并/部署**。工作区 `.worktrees/chat-domain` / `xiwanzi/ysm-admin-integration`，后台新增统一登录的 `/admin?section=ysm`；仅 Web/Go 候选，原游戏模型与插件未改。见[验收](qa/ysm-admin-integration.md)、[接入契约](contracts/ysm-admin.md)、[上线步骤](deployment/ysm-admin-candidate.md)。现有发布仍以下方记录为准。

2026-09-26：**App 2.0.14（21400）与配套 Go 已于 20:00 上线**（PR #48，源码 `4e95ae0df69b`）。在线名单接口、正式安装包、签名、公网更新及四节点重连验证通过；Web 2.0.19、游戏插件和原业务数据保留。维护 `.worktrees/chat-domain` / `xiwanzi/release-v214-live`。见[部署记录](deployment/deuterium-2.0.14-live.md)，下方候选说明为早期状态。

2026-09-26：**聊天在线名单与连续返回修复为本机候选，尚未推送/部署**。基于最新 `origin/main` `cbc522d14ba5`，Android 与线上 2.0.13（21300）源码基线一致；维护 `.worktrees/chat-domain` / `xiwanzi/chat-online-back-fix`。配套 Go 在线快照接口、App 弹窗刷新和返回回调已实现，构建/单元/集成/模拟器事件验证通过；发布和用户真机側滑待后续。见[验收记录](qa/chat-online-back-fix.md)。

2026-09-19：**App 宣传页 https://app.deuteriumix.com 已于 21:36 上线**。沿用第七版设计，删除未接正式服务等旧说明；下载和网页入口均为“敬请期待”，本域名不提供 APK。独立静态站与 HTTPS 自动续期验证通过，见[上线记录](deployment/app-landing-live.md)。

2026-09-19：**玩家社区域名 https://chat.deuteriumix.com 已于 21:12 上线**。独立 HTTPS 证书和自动续期验证通过，Go 精确兼容新旧来源，App 2.0.13 / Web 2.0.19 保持；旧 IP 接口和启动器入口保留。源码 `a1064819ac0a`，维护 `.worktrees/chat-domain`，见[上线记录](deployment/chat-domain-live.md)。下方为较早记录。

2026-09-15：**Go 并发优化已于 20:47:45 上线**（PR #43，源码 `a824088cb518`）。请求分组限额、公共聊天读取合并、AI/密码短时等待与管理员容量接口已部署。App 2.0.13、Web 2.0.19、启动器 0.6.1、申请站 1.0.2 保持；配置、71 条账号记录、27 笔订单和 5 个委托保留。按用户要求未运行测试或压测。维护工作区 `C:/DeiteriumAPP-IX/.worktrees/backend-concurrency`，见[上线记录](deployment/backend-concurrency-live.md)。下方为较早记录。

2026-09-13：**申请站 1.0.2 配色优化已于 20:48 上线**（PR #41）。主页面减少装饰性亮黄，弹窗查询改深色按钮、返回改描边按钮；仅替换静态 CSS，申请、审核、邮件和其他组件不变。见[上线记录](deployment/admission-1.0.2-live.md)。下方为较早记录。

2026-09-13：**Web 2.0.19、配套 Go 与启动器 0.6.1 已上线**（PR #39）。管理页新增程序签名发布，继续使用现有 OSS；启动器内嵌玩家社区并接入独立更新器，实际自动升级通过。App 保持 2.0.13，业务与白名单状态保留。见[上线记录](deployment/dlauncher-0.6.1-live.md)。下方为较早记录。

2026-09-13：**审核邮件配色保护与小尺寸徽章已于 10:47 上线**（PR #37）。通过/拒绝模板均采用已确认的锁色处理，内嵌图标从 1.82 MB 降至 32.5 KB，仍使用 CID 附件。SMTP 配置及审批流程保留，Web 仍为 2.0.18、App 2.0.13。见[上线记录](deployment/mail-appearance-live.md)。下方为较早记录。

2026-09-13：**Web 2.0.18、配套 Go 与白名单审核邮件已于 02:05 上线**（PR #35）。通过/拒绝结果自动发至申请人的 QQ 邮箱，使用后台已保存的 SMTP 配置；两种模板实际测试均被接收。支持模板预览、测试、投递状态与失败重试，过期通知取消。App 仍为 2.0.13，70 条已撤销资格未恢复。见[上线记录](deployment/admission-review-email-live.md)。下方为较早记录。

2026-09-13：**SMTP 队列修复已于 00:36 上线**，解决 MariaDB 10.5 不支持 `SKIP LOCKED` 导致测试邮件始终待发送的问题。两封已排队测试邮件已被 QQ SMTP 接收，配置/授权码保留。Web 仍为 2.0.17，App 2.0.13，70 条已撤销白名单资格未恢复；审批通过邮件尚待正文确定。见[上线记录](deployment/smtp-queue-fix-live.md)。下方为较早记录。

2026-09-13：**Web 2.0.17 已于 00:10 上线**，修复白名单列表切回申请记录导致全屏白屏。按用户最新要求，00:14 已撤销全部 70 个现有白名单资格，当时有效资格为 0；账号及历史保留，在线连接未主动断开。以后须重新申请审批，禁止自动恢复旧导入资格。App、Go、Gateway 沿用原版。见[部署与重置记录](deployment/web-2.0.17-live.md)。下方为较早记录。

2026-09-12：**Web 2.0.16、通行申请站和 Velocity Gateway 1.0.0 已发布**；配套 Go 源码经 PR #28、#29 合并，App 保持 2.0.13。独立申请站暂用 [IP 入口](https://47.103.99.34/admission/)，后台支持审批/手动添加/移除/操作历史。默认服务器 Amiya，已有玩家重连优先返回上次子服。70 个旧有效账号通行资格及原业务数据保留，四个 Core 节点在线。维护工作区 `.worktrees/admission-velocity`。详见[部署记录](deployment/admission-v1-live.md)，下方为较早状态快照。

2026-09-12：**App 2.0.13（21300）、Web 2.0.15 与配套 Go 已于 19:15 上线**，源码 `15690c138288`（PR #26），xiwanzi 更新提醒已推送。工作台支持当前管理员密码确认的永久注销，App/Web 同步清理关联展示，历史交易匿名保留；已部署启动器 0.5 源码一并合入，内容版本 5 和配置未改。71 个账号、27 笔订单、5 个委托保留，四节点在线；本轮未注销真实账号。维护工作区 `.worktrees/account-erasure`，分支 `xiwanzi/release-v213-live`。见[部署记录](deployment/deuterium-2.0.13-live.md)。下方为较早记录。

2026-09-12：**管理员永久注销账号候选已完成本机实现与验收，尚未推送/部署**。源码 `038718b068fb`，工作区 `.worktrees/account-erasure`、分支 `xiwanzi/account-erasure`。工作台校验当前管理员密码，后端清理账号与展示数据，App/Web 同步清理缓存，历史交易匿名保留。没有操作真实旧号；线上仍为 2.0.12。见[验收记录](qa/account-permanent-deletion.md)。

2026-09-12：**DLauncher 0.5.0 / Web 2.0.14 配套 Go 已上线**，切换为整包发布后仅 McPatch 更新，删除首次安装接口和配置；Deuterium IX 使用 D9 徽章及 DSA 素材，其他入口隐藏。本轮从 origin/main 4a9c249 继续，保留并上线同步超时终态修复。维护 `.worktrees/dlauncher-bundled` / `feat/dlauncher-bundled`；用户要求本轮不生成完整游戏包。见[部署记录](deployment/dlauncher-0.5-live.md)。

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
