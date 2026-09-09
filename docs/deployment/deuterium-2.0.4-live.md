# Deuterium 2.0.4 部署记录

2026-09-09（UTC+8）。**App 2.0.4（20401）、Web 2.0.4、配套 Go、Core 1.0.1 与 XConomy `.2` 已部署。** 12:26 在线更新清单发布，xiwanzi 账号已有唯一更新通知。源码通过 [PR #3](https://github.com/xiwanzi/DeiteriumAPP-IX/pull/3) 合入公开维护仓库。

## 安装与源码

- [APK 下载](https://47.103.99.34/downloads/Deuterium-2.0.4-20401-test.apk)，23,319,339 bytes，SHA-256 `6af287aff417242cd2ac9fb3d7c7b9f66d876b5f48d60a7d2560e2ca433ee96f`。
- 包名 `com.deuterium.app.uilab`，Android 8.0+，原测试签名 SHA-256 `04a213066a78a968c2a1328a153a0967f99d117e0a826285e3f4af705db6f2f3`，支持覆盖安装。
- App/Go/Core/XConomy 源码 `812f1d1a50566b8d2ac266d8219ba74fb73a9211`。网页静态版本元信息另由 `558156655b56bba70aefdab902598f923b93ba48` 修正，业务 JS/CSS 与 PR #3 相同。
- 当前维护工作区：`C:/DeiteriumAPP-IX/.worktrees/release-v204`。本机交付：`C:/DeuteriumAPP/delivery/Deuterium-2.0.4/`。此前 20400 是候选，20402 是隔离安装夹具，均不是在线发布版本。

| 对象 | 当前文件/目录 | SHA-256 |
| --- | --- | --- |
| Go | `/opt/deuterium/releases/app-2.0.4-812f1d1a5056/deuterium` | `5d4e48cfec9f1074ad417606bc843342469f3196316256b7569711558ee2ec84` |
| Web | `/var/www/deuterium/releases/web-2.0.4-812f1d1a5056/dist` | ZIP `887c2c4846bc52ec041f9cf5b6ed4681083c72d1da792809422a1e678862eed8` |
| Core | 四服 `Deuterium-Core-1.0.1.jar` | `2cfc69580687759bafa1654a177c859426391d82c71ab99df47188153c0cd2bc` |
| XConomy | 四服 `XConomy-Bukkit-2.26.3-deuterium.2.jar` | `abcdf0337a227ecef55b4e4c1e4f9c13d477e760cb8cd1c32cd42c6449229fd5` |

## 已上线行为

公告可永久删除正文、草稿、发布版本及关联通知，保留删除审计事实。平台介入有待跟进数量、SMTP 配置和持久提醒队列。管理审计可查全服流水及每名玩家的订单、市场与官方商品，包含个人隐藏/下架/售罄记录。商品编辑提供分组表单、单一选项自动选中、保存并发布与退出保护。

App 删除按钮统一红色语义，订单/委托/商品/流水按时间倒序；最近列表、日统计及游戏内收入/支出共用 XConomy 已提交账本。转账成功遇到旧查询在途时补刷新，避免最新交易显示延迟。主题切换保留 Activity 和页面状态。

安装器成功打开后自动退出旧 App 任务，保留安装器读取 APK 的权限；新包收到系统安装完成广播时也处理旧版残留任务。已经打开的新版界面不会被迟到通知关闭。正常窗口显示可解除启动动画等待，修复更新后卡在加载页。安装完成、Open、桌面重开及取消安装已在模拟器走通，见[安装回归](../qa/install-v204-2026-09-09.md)。

SMTP 密钥已在服务器私有环境中生成并备份，管理页 `secretStorageReady=true`。**实际 SMTP 主机、发件授权码和接收邮箱尚未配置，邮件提醒目前未启用。** 管理员可在“官方管理 → 邮件提醒”填写、保存并发送测试邮件。没有代填收件人或向外部邮箱发送真实测试邮件。

## 验证

- 公网完整下载 APK 的字节数、摘要与本机构建一致；20004、20100、20101、20200、20300、20400 可检测到 20401，20401 和较新的 20402 不提示重复安装或降级。
- 网站 HTML 引用新构建，公网 JS/CSS 与交付 ZIP 逐字节摘要相同。Nginx 原本固定返回 2.0.0 的运行配置已改为读取当前静态 `web-config.json`，避免后续版本继续与页面不一致。
- 三个常驻节点运行正常、游戏协议探测成功，加载日志确认 Core 1.0.1、XConomy `.2`，Core 主动连接正常；MEK 保持停止和禁领。Core 原桥接状态中的 `version` 仍为既有固定 `1.0.0` 文本，本次安装版本以 JAR 摘要及加载日志为证，未把该字段作为安装证明。
- 使用临时、随后撤销的 QA 会话只读验证生产接口：本人首屏 25 条账本含游戏内记录，时间倒序、详情和当日汇总可读；管理端能读到全服账本、游戏来源、8 条订单及 5 件商品。读取账本后再次查询余额正常，实际 XConomy 连接池未被只读事务状态影响。既有 8 条订单、3 条委托保留，没有创建测试付款或删除真实公告。
- 121 项 Android 单测、Lint 0 errors / 29 warnings；原 2.0.4 的 Go 123 项顶层 race/integration、Web 54 项测试与 18 项浏览器流程、Core 29 项、XConomy 10 项真实数据库测试通过；本次安装增量补做真实系统安装器和旧版覆盖回归。详细边界见[功能 QA](../qa/admin-ledger-v204-2026-09-09.md)。

账号通知事件 `app.release:20401` 仅 xiwanzi 一条，未广播公共聊天。手机实际收到系统弹窗、真机手感和真实游戏客户端领取/切服不冒充本次已测。

## 备份、迁移与恢复

云端备份：`/var/backups/deuterium/app-2.0.4-20260909-20401`，保存原配置、环境、更新清单、程序和网站指向、Nginx 配置及后端 SQL；SQL gzip 170,155 bytes，SHA-256 `fa584700218623e51c63d2b74cb3c6213f6643f8afc498273775d2c38871051e`。SMTP 密钥所在的新环境文件也在该私有备份中。

游戏侧备份：`E:/Deuterium_IX/deployment-backups/app-v204-20260909-20401`。12:11 正常停服并确认全部原 Java/包装进程退出，12:13 完成三库一致性快照和四服配置/插件/玩家状态备份，12:16 换包，12:17 三服完成启动。三库 SQL gzip 175,508 bytes，SHA-256 `c21e90b576cf0165c74eb27e356b913baaa196735c20655949c6d11ffcb55ca5`；四个 ZIP 均通过 CRC 和逐文件摘要。配置摘要前后一致，Mail、Bridge、Sync 与客户端 Mod 未更换。

15 个既有迁移摘要与源码逐一匹配。新增 `020_admin_email_v204.sql` SHA-256 `eb0fbd39f46f357fab2e89abe34c656ce76f0437d2447131ca491f1efac68d7f`，仅增加邮件配置/队列表；XConomy 增加账本查询索引，不改原始账本和余额。

启动轮询曾遇到一次管理器过渡状态的 `JSONDecodeError`；三条启动 ACK 均已持久保存，之后通过只读进程、端口、加载日志和后端连接确认成功，没有重复发送启动命令。所有旧 JAR、原 Go/Web 目录和私有备份保留。

如需回退，只切旧程序/配置/更新清单；保留新增表、索引与新业务，不恢复旧 SQL 覆盖新交易。旧 Go 仍兼容新插件的原资金 API，但不提供新的统一流水和管理能力；App/Web 须随功能依赖协调回退。

[公网验证](artifacts/v204/public-verification.json)、[生产只读验证（已去除金额）](artifacts/v204/live-readonly-verification.json)、[云端启用](artifacts/v204/cloud-activation.json)、[游戏备份](artifacts/v204/minecraft-backup.json)、[启动确认](artifacts/v204/minecraft-start-verified.json)、[构建摘要](artifacts/v204/release-build.json)。SQL、配置秘密和私钥不在公开下载或源码中。
