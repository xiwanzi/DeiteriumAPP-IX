# Deuterium 2.0.4 候选交付

2026-09-09。**功能代码和本机候选包已完成，尚未推送、合并或部署。** 线上仍以 2.0.3 部署记录为准。代码提交 `018d1e25b0802a5b5130aa3eff4a04b0a612fd73`，本地分支 `feat/admin-ledger-v204`。

## 本次变化

- 网页公告增加永久删除及确认；同步删除公告正文、草稿、发布版本和关联通知，保留删除审计事实。
- 新增 SMTP 配置、多个接收邮箱、测试邮件、送达状态；平台介入新申请和状态变化进入持久邮件队列，失败重试。管理入口显示待跟进案件数量，邮件链接直达案件。
- 管理审计可查全服玩家流水，按玩家查看所有买入/卖出及管理店铺的订单、市场与官方商品，包含个人隐藏、售罄和下架记录。
- 官方商品编辑按内容/售价交付分组，单一品牌、分类、模板自动选中；提供固定的“保存草稿/保存并发布”，保留原请求键进行重试，防止误触关闭。玩家商品也有未保存内容保护。
- App 删除按钮统一为淡红底、红色图标和文字；订单、委托、商品和流水按真实时间倒序。主题切换处理 `uiMode`，保留 Activity 和页面状态。
- 游戏内消费、收入、转账与 App 交易读取同一份已提交的 XConomy 账本；游戏内来源标为“游戏内收入/游戏内支出”。最近列表只读第一页，历史账单独立按日期分页；本日收支由账本按北京时间汇总。转账成功期间遇到旧查询在途，会补一次刷新。

## 候选文件

交付目录：`C:/DeuteriumAPP/delivery/Deuterium-2.0.4-candidate/`。

| 文件 | 用途 |
| --- | --- |
| [Deuterium-2.0.4-20400-test.apk](C:/DeuteriumAPP/delivery/Deuterium-2.0.4-candidate/Deuterium-2.0.4-20400-test.apk) | App 2.0.4（20400），Android 8.0+，包名 `com.deuterium.app.uilab` |
| [Deuterium-Web-2.0.4.zip](C:/DeuteriumAPP/delivery/Deuterium-2.0.4-candidate/Deuterium-Web-2.0.4.zip) | 构建后的静态网站 |
| [deuterium-linux-amd64](C:/DeuteriumAPP/delivery/Deuterium-2.0.4-candidate/deuterium-linux-amd64) | Linux amd64 Go 服务，CGO 关闭 |
| [Deuterium-Core-1.0.1.jar](C:/DeuteriumAPP/delivery/Deuterium-2.0.4-candidate/Deuterium-Core-1.0.1.jar) | Core 新增只读流水命令 |
| [XConomy-Bukkit-2.26.3-deuterium.2.jar](C:/DeuteriumAPP/delivery/Deuterium-2.0.4-candidate/XConomy-Bukkit-2.26.3-deuterium.2.jar) | 经济账本查询与当日汇总 |
| [对应 XConomy 源码与许可](C:/DeuteriumAPP/delivery/Deuterium-2.0.4-candidate/XConomy-2.26.3-deuterium.2-source.zip) | 扩展源码、构建脚本、上游源码和 GPL 许可 |
| [manifest.json](C:/DeuteriumAPP/delivery/Deuterium-2.0.4-candidate/manifest.json) | 所有文件的字节数、SHA-256、源码提交和签名信息 |

APK 沿用 2.0.3 的测试签名：`04a213066a78a968c2a1328a153a0967f99d117e0a826285e3f4af705db6f2f3`，已在模拟器覆盖安装。**App 和 Go 的统一流水依赖新 Core/XConomy，需配套部署后用于真实账号。** 网站和 APK 均未写入线上更新清单，未发送账号发布提醒。

## 上线准备与顺序

1. 确认源码推送范围。GitHub CLI 本轮实测维护仓库 `xiwanzi/DeiteriumAPP-IX` 为 `PUBLIC`，与原工程约定的私有维护入口不一致；本轮未找到此前明确公开此增量源码的授权，因此候选先保留本机。
2. 备份现有 Go/Web/插件、服务配置与环境、后端和 XConomy 数据库；保存当前版本文件及更新清单。
3. 按既有四服统一停服流程替换 XConomy `.2` 和 Core `1.0.1`，维持原经济权威节点配置、MEK 停止及禁领状态。XConomy 自动检查并添加两个查询索引，不改旧经济记录。
4. Go 执行新增迁移 `020_admin_email_v204.sql`，再切换新二进制并检查健康。迁移 SHA-256：`eb0fbd39f46f357fab2e89abe34c656ce76f0437d2447131ca491f1efac68d7f`。既有迁移未修改。
5. 在服务私有环境文件配置 `DEUTERIUM_SMTP_KEY`：32 个随机字节的 Base64，权限与现有服务秘密一致，并随备份保管。该密钥不填进网页，也不进入仓库。
6. 部署静态网站，在“官方管理 → 邮件提醒”填 SMTP 主机、端口、TLS/STARTTLS、发件邮箱/授权码及接收邮箱，保存并发送测试邮件。当前尚未配置真实邮箱服务；本机验证使用隔离 SMTP。
7. 只读检查真实账户流水、历史订单、待办案件及邮件队列。确认配套运行后再发布 APK/在线更新；涉及真实邮件发送和正式账户提醒按发布授权执行。

失败回退只切旧程序与配置，保留新增表、索引及新产生的业务数据，不用旧 SQL 覆盖新记录。旧 Core 不支持新查询，因此 App/Go/插件须按配套关系回退。公告的永久删除不会因程序回退恢复。

## 验证与限制

见 [本次 QA](../qa/admin-ledger-v204-2026-09-09.md)、[文件摘要](../qa/artifacts/v204/manifest.json)、[接口与数据规则](../contracts/admin-ledger-v204.md)。真机手感、实服重启后查询、真实邮箱收件和正式更新发布尚未执行，不能用本机通过代替上线验证。
