# App 2.0.15、社区 2.0.21 与邮件账号后端上线

2026-10-04（UTC+8）：**09:28:32** 完成 Go、数据库迁移与社区 **Web 2.0.21** 切换；**09:30:23** 开放 **App 2.0.15（21500）** 更新。功能通过 [PR #53](https://github.com/xiwanzi/DeiteriumAPP-IX/pull/53) 合入 main，部署源码 `6d5a2fdde8f15ce172c00b9b6fe170745e6ffe6d`。

[玩家社区](https://chat.deuteriumix.com/) · [邮件系统](https://chat.deuteriumix.com/admin?section=email) · [工单工作台](https://chat.deuteriumix.com/admin?section=support) · [App 下载](https://47.103.99.34/downloads/Deuterium-2.0.15-21500-test.apk)

## 发布内容与当前可用范围

- App / Web 改为邮箱验证码注册真实 Deuterium ID，进入账号后单独核验 QQ 和游戏身份；旧账号默认已验证。新增独立中文昵称、邮箱找回密码、登录后的密码修改和邮件客服入口。
- 后台新增邮件系统与邮件工单：客服邮箱收件建单、日期加四位随机数字编号、DM 收件回执、按邮件线程归并、附件下载、人工邮件回复、内部备注及客服专用权限。原 `platform.admin` 继续作为超级管理员；新工单权限不授予账号、财务等后台权限。
- 三套验证码品牌模板及预览已随 Go 上线。Help 地址只保留 `https://help.deuteriumix.com` 链接，本轮没有建设 Help 网站。
- **DM、客服 SMTP/IMAP、内部转交邮箱尚无授权码，保持未启用。** 后台已预填用户指定地址及杭州 DM / 企业邮箱服务器。邮箱新注册当前明确返回 `MAIL_NOT_READY`，不创建账号、验证码或邮件；工单收发也须配置通道后启用。不能把代码上线表述为真实邮箱已接通。
- 首次保存 DM 设置之前，原白名单审核邮件及管理通知继续走已配置的旧 SMTP。保存 DM 后改为该通道，显式关闭不会自动回退旧发件地址。既有 SMTP 密文、通知开关及收件人配置保留。**验证码与工单邮件不会借用旧 SMTP。**
- 游戏 Core / Gateway / Mail / XConomy / Sync 未更换，游戏服务未重启。旧 IP、新社区域名、启动器、模型管理、白名单保持。

## 数据与迁移

只增加 `032_mail_identity_support.sql`，旧 27 个迁移的摘要与生产完全一致。新增邮箱身份字段、邮件队列及工单表；当前七个账号均为 `game_verified=true`。

发布前、停稳后和迁移后核对原身份字段、账号/业务 ID 集合、全部 `commerce_*` 表、权限/别名、注销记录、申请/白名单/操作记录、SMTP 配置及启动器设置。**7 个账号、27 笔订单、5 个委托、0 条注销记录完整保留**；没有恢复此前抹除的账号。启动器配置/发布版本仍为 9，服务及环境文件保持原摘要。

首次迁移尝试因日常服务数据库账号仅有数据读写权限、无 CREATE / ALTER 权限而失败；自动恢复旧服务。检查确认该尝试未新增字段或表、未写入迁移记录。随后通过服务器既有 root socket 身份执行迁移，再以原受限服务身份运行新 Go，没有扩大服务数据库权限。

## 产物

| 产物 | 字节数 | SHA-256 |
| --- | ---: | --- |
| App 2.0.15 / 21500 | 8,891,922 | `fd675f0587f99e06f0e7f1004888b87b4ea10f93ffee7d3e4280cad90711574c` |
| Go Linux amd64 | 29,805,957 | `215fc855e80d8e5189f8b3c1ddab743ed134b48d405c43b37abf1b4a6ba44ea9` |
| Web 2.0.21 ZIP | 33,733,811 | `440441011baa0bf4658b20cddae8a38c3fce8a9899b4ccce4a02921f6b6bf259` |

Android 包名 `com.deuterium.app.uilab`，沿用非 debuggable performance、R8、资源压缩和 Baseline Profile；证书 SHA-256 `04a213066a78a968c2a1328a153a0967f99d117e0a826285e3f4af705db6f2f3` 与已发布版本一致。Go 1.27.1 / CGO=0 从干净合并源码构建，`vcs.modified=false`。Android/Web 构建来源 `9497f7cf459f`，与合并版本产品源码一致。

- Go：`/opt/deuterium/releases/release-2.0.15-6d5a2fdde8f1`。
- Web：`/var/www/deuterium/releases/web-2.0.21-6d5a2fdde8f1/dist`。
- 本机交付：`delivery/Deuterium-2.0.15/`，含 APK、Go、Web、mapping 和 manifest。
- 维护工作区 `C:/DeiteriumAPP-IX/.worktrees/mail-identity-support`，分支 `xiwanzi/release-v215-live`。

## 本轮验证

- Go 全量普通测试及 vet 通过；新邮箱身份、邮件工单、权限、原审核邮件以及本次新增旧 SMTP 过渡逻辑的集成 race 验证通过（25.543 秒）。此前候选阶段完整证据见[邮件账号 QA](../qa/mail-identity-support.md)与[模板 QA](../qa/verification-mail-templates-20261004.md)，不冒充本次重复全量集成测试。
- Web 71 项测试、生产构建通过。Android 152 项 JVM 测试、正式包构建、lint 通过（0 errors / 23 warnings）。
- Android 15 模拟器从原 2.0.10 同签名覆盖到 2.0.15 成功；使用正式压缩包及独立测试 APK 验证邮箱验证码请求不含密码、注册密码验证、找回密码浅深色、玻璃关闭、键盘及返回行为。测试请求在设备本地拦截，无真实账号或邮件写入。真机手感与实际邮箱收信仍待配置后验收。
- 公网 IP / 社区域名 TLS、Web 2.0.21、App 更新接口及 APK 大小/摘要验证通过。20004、21300、21400、21499 获得 21500；21500 和 21501 不重复更新或降级，原 15 个发布清单条目保留。
- 真实后台短期会话检查旧账号仍已验证、邮件配置入口、工单入口及三套模板成功；匿名访问 401，普通玩家访问后台 403。未配置的注册请求正确返回 503 `MAIL_NOT_READY`，没有创建测试账号或待发邮件。所有临时会话已撤销。
- 真实浏览器查看邮件总览、模板预览、工单空队列与 390px 窄屏；没有页面脚本错误，窄屏无横向溢出。现有生产工单为 0，本次未伪造工单、未发送真实邮件。
- 最终运行进程文件摘要与候选 Go 一致，服务 active / ready；Login、Amiya、Odyssey、MEK 四个 Core 节点均在线，聊天在线接口可用。

## 备份与恢复

备份目录 `/var/backups/deuterium/release-2.0.15-20261004-6d5a2fdde8f1`（0700、敏感文件 0600）。备份只包含本次清理后的当前数据库，不导入旧账号数据。

- 发布前一致性备份：19,699,165 字节，SHA-256 `9de0200982d5b13751f985ec8ffd4abfdb551c52197f9411b70c7555441b989a`。
- 成功切换前停稳备份 `database-stopped.sql.gz`：SHA-256 `8cab51a791c5da447091c5c9e8a1a62cb6ee7d3b453f1abe66b584568854f45f`。两份备份均完成解压校验。
- 原 Go：`/opt/deuterium/releases/model-management-918584f72edf`；原 Web：`/var/www/deuterium/releases/web-2.0.20-25931e2ce897/dist`。原配置、环境文件及 App 清单保留备份。
- 如需停止分发可恢复本轮备份的 App 清单并重启当前 Go；已安装 21500 的设备不能自动降级，应发布更高版本修复。
- **已有邮箱待验证账号后，不可直接切回不理解 nullable UUID / 验证状态的旧 Go；优先前向修复。** 迁移后的数据库不能用历史整库覆盖，更不能恢复已抹除的账号。邮件故障通过暂停通道、保留队列和游标处理。

[产物清单](artifacts/v215/manifest.json) · [预检](artifacts/v215/preflight.json) · [备份/上传](artifacts/v215/stage.json) · [切换](artifacts/v215/activated.json) · [App 发布](artifacts/v215/published.json) · [公网验证](artifacts/v215/public-verification.json) · [接口验证](artifacts/v215/account-verification.json) · [最终状态](artifacts/v215/final-state.json) · [浏览器与设备验证](artifacts/v215/browser-verification.json)
