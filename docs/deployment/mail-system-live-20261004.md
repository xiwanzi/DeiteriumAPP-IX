# 邮件系统接通与样式上线

2026-10-04（UTC+8）。**12:21:30** 发布邮件样式后端，**12:22:49** 接通 DM、客服企业邮箱、内部系统邮箱及客服 IMAP；真实联调后，**12:49:51** 发布图片与回执修正。App 仍为 **2.0.15（21500）**，社区仍为 **Web 2.0.21**，游戏插件未更换。

[邮件系统](https://chat.deuteriumix.com/admin?section=email) · [工单工作台](https://chat.deuteriumix.com/admin?section=support)

## 来源与范围

- [PR #55](https://github.com/xiwanzi/DeiteriumAPP-IX/pull/55)，首次邮件后端源码 `c8ad1b52d8903e137eb57beea46fe64e8ef8ab2f`。
- [PR #56](https://github.com/xiwanzi/DeiteriumAPP-IX/pull/56)，最终源码 **`b0b6686cf1adb2e9c3a97f01baa3d1039bbb99af`**。
- 工作区 `C:/DeiteriumAPP-IX/.worktrees/mail-identity-support`，记录分支 `xiwanzi/mail-live`。
- 中文使用官网 HarmonyOS Sans SC，普通数字使用 Novecento Sans Wide；六位验证码按用户指定保留原 Consolas 等宽数字。徽章 48px、官网字标、12px 联系说明与 11px 版权统一执行。
- 人工回复保留自由正文、写入时的客服署名和品牌页脚。**按用户最新要求，不针对 Gmail 自动折叠修改署名或页脚布局，不增加回复时间等绕过折叠的字段。**
- 自动收件回执保留工单编号、收件确认和回复说明，**没有问题摘要或「待审查」展示**。新回执队列也不再复制来信正文；工单本身仍保留原始来信。

## 已接通通道

| 用途 | 地址 | 连接 |
| --- | --- | --- |
| 验证码、审核通知、自动回执 | `noreply@notify.deuteriumix.com` | 杭州 DM，`smtpdm.aliyun.com:465` TLS |
| 人工客服发信 | `support@deuteriumix.com` | `smtp.qiye.aliyun.com:465` TLS |
| 客服收件 | `support@deuteriumix.com` | `imap.qiye.aliyun.com:993` TLS，INBOX 只读同步 |
| 内部 Help 邮件转交 | `go@deuteriumix.com` | `smtp.qiye.aliyun.com:465` TLS |

三个 SMTP 账号均完成服务器端 TLS 1.3 认证，客服 IMAP 认证及只读打开 INBOX 成功。通过既有管理 API 保存配置，使用既有密钥加密；凭据未写入源码、发布记录或公开工件。配置版本均为 1，后端无需新增环境变量。

收件启用时间为 `2026-10-04T04:22:49.890899339Z`，`importExisting=false`。原有两封邮件按启用时间跳过，没有导入或发送历史回执。原 SMTP 配置、通知开关和管理收件人保留；自动通知现按已启用的 DM 通道发送。

`go` 到 `support` 的内部 SMTP 传输已实测，客服 IMAP 找到同一 Message-ID。该探测带自动邮件标记，正确被忽略，没有创建客户工单。**Help 网站仍未建设**；内部邮箱连通不等于该网页已上线。

## 真实联调与修正

用户指定并确认了一个 Gmail 测试收件地址，本文不公开完整地址。

1. 通过真实注册验证码接口发送联调验证码，用户确认收到；未完成注册，也没有创建 `MailCheckOct04` 测试账号。
2. 企业邮箱发送客服联调邮件，用户确认收到并直接回复。IMAP 收到来信，创建工单 **202610042852**，DM 自动回执及工单人工回复均已发送，用户确认两封后续邮件也收到。该测试工单已解决。
3. 用户反馈 CID 徽章及字标在 Gmail 回复时被携带为图片附件；同时要求去掉自动回执中的问题摘要。
4. 所有邮件品牌图改为官网 OSS 的公开 HTTPS 资源，使用内容摘要命名与一年 immutable 缓存。三份 PNG 匿名下载后的大小与 SHA-256 均匹配源文件；不包含收件人或工单跟踪参数。**发送的邮件只有纯文本和 HTML 两个 MIME 备选部分，不附带品牌图片。**
5. 修正后再次发出独立联调邮件。用户确认「已回复，没有图标附件」；新工单 **202610045235** 的实际收件附件数为 **0**，自动回执与最终人工确认邮件均被 SMTP 接收，该测试工单已解决。

Gmail 对重复签名的折叠保留客户端原行为，本轮没有承诺强制关闭。旧邮件中已经存在的 CID 附件不会被远程移除；修正作用于新发送的邮件。

DM 最终投递事件分发尚未接入；后台 `SENT` 仍表示发信服务器已接收。上述用户明确确认的收信结果属于真实联调证据，不把所有 `SENT` 自动等同于进入收件箱。

## 构建、服务与数据

| 阶段 | Go 字节数 | SHA-256 |
| --- | ---: | --- |
| 首次接通版本 | 29,850,890 | `c5ecca39479ecab99997537ba110f0fea4cded76422a9a0197261f209e0e5060` |
| 最终修正版本 | 29,805,477 | `417c287221171298d805c8c8cc5e074c574fb397d7651b5d3da5ffc6f47b6b8e` |

当前程序 `/opt/deuterium/releases/mail-fix-b0b6686cf1ad/deuterium`，Linux amd64、Go 1.27.1、CGO=0，从干净合并源码构建。运行进程摘要核对一致，服务 active / ready；Login、Amiya、Odyssey、MEK 四个 Core 节点在线。

- 本轮没有数据库迁移，28 个迁移的摘要保持一致。App 清单、Web 目录、服务文件、原环境文件与启动器配置保留。
- 两次切换时均为 7 个账号、27 笔订单、5 个委托、0 条注销记录；既有业务 ID 全部保留。
- 启用邮箱注册后正常新增 1 个邮箱账号，最后核对为 **8 个账号，其中原 7 个已验证、新账号待游戏验证**。不是恢复已抹除账号，也不是联调脚本创建的账号。27 笔订单、5 个委托和 0 条注销记录保持。
- 两个指定测试工单已解决；其余正常来信保持业务状态，未作为测试记录清除。

验证：首次发布全量普通 Go 测试与 vet 通过；此前字体和邮件专项 race 通过。修正轮通知/API/存储测试、vet、工单/企业邮箱/Help 专项 race 通过（14.338 秒），MIME 校验确认为无图片附件；预览中 HTTPS 图像实际加载、回执无摘要、手机无横向溢出。所有临时维护会话均撤销。

## 备份与回退

- 首次：`/var/backups/deuterium/mail-live-20261004-c8ad1b52d890`，数据库备份 19,803,628 字节，SHA-256 `0282b78e9bbf7cac6896a0e339481d24e23343dc40d470992d5397b525ef053b`。
- 修正轮：`/var/backups/deuterium/mail-fix-20261004-b0b6686cf1ad`，数据库备份 19,825,854 字节，SHA-256 `668dd8b258e9ac5f187c813d2a368843184f775116cfc0d1c8f9dd425b32edee`。
- 目录 0700、敏感文件 0600，压缩备份解压验证通过。备份不用于覆盖正常新增账号、工单或恢复已抹除账号。
- 必要时可正常停止 Go 后切回首次接通版本 `/opt/deuterium/releases/mail-live-c8ad1b52d890`；账号结构兼容，但会恢复旧 CID 品牌图及摘要行为，优先前向修复。不要回退邮件配置或整库。
- 已发送邮件依赖的公开品牌图片必须保留；新资源更换内容时使用新文件名，不能覆盖或删除旧版本。

[最终构建清单](artifacts/mail-live/final-manifest.json) · [配置结果](artifacts/mail-live/configured.json) · [真实工单联调](artifacts/mail-live/roundtrip-final.json) · [最终服务核对](artifacts/mail-live/live-verification.json) · [公开图片清单](artifacts/mail-live/assets.json) · [最后业务状态](artifacts/mail-live/final-business.json)
