# 白名单审核邮件上线

2026-09-13 **02:03** 部署 Web **2.0.18**、配套 Go 与申请站 **1.0.1**，**02:05（UTC+8）** 开启白名单审核邮件。源码 [PR #35](https://github.com/xiwanzi/DeiteriumAPP-IX/pull/35) 已合并，提交 `57878e526653dfbef157c696c3b304a1cf6a25e0`。App 保持 2.0.13，游戏插件未更换。

通过/拒绝审批与邮件入队在同一事务完成，每份申请只有一条结果任务。正式邮件使用已确认的通过 v5、拒绝 v3 版式及文案，发送至申请人填写的 `QQ号码@qq.com`；拒绝原因取本次实际审核原因，审批通过的内部备注不进入邮件。

[邮件管理](https://47.103.99.34/admin?section=email) 现有 SMTP 信息与加密授权码保留，设置版本从 8 升至 9，仅开启审核结果邮件。支持两套模板预览/测试，管理测试发送至配置的管理收件邮箱；投递记录展示类型、申请人、收件地址、尝试次数及状态。

失败沿原任务和 Message-ID 重试；已撤销或被后续申请取代的旧通知取消。关闭审核提醒时的新审核为 `SKIPPED`，不在重新开启时补发；已排队的审核邮件暂停。总开关关闭会暂停所有邮件。SMTP 接收确认丢失仍可能导致收件端重复，固定编号不等于协议层恰好一次。

手动添加、移除和旧名单导入不触发审批邮件。未补发上线前的历史审核；原 70 条撤销资格保持不变，账号及订单/委托记录保留。

验证：完整 Go `-race -tags integration ./...` 与 vet 通过；Web 68 项测试和生产构建通过。实际 MariaDB 10.5.29 的随机隔离库通过 5 项 SMTP/审核集成用例，临时数据库及受限测试账号均已移除。浏览器完成设置保存、模板预览与测试、拒绝、重新申请通过、收件隔离及真实原因转义。补测 HTML SMTP 的 TLS/STARTTLS 及旧拒绝通知取消通过。

上线后通过后台测试接口入队两封 HTML 邮件，均于 02:04:56–58 被 QQ SMTP 接收，`attempts=1`。正式审批没有使用真实玩家做测试，临时验收会话已撤销。实际运行二进制和公网 HTML 与发布产物一致，网关心跳正常。

| 产物 | SHA-256 |
|---|---|
| Go | `fc85f44d9332854d5ef4ee3709086e344bb6bb825d4e14523ddaf51fa27338b1` |
| Web 2.0.18 ZIP | `a29c4dc85f5d2c8f7f5f7897490c1a0608049936e175e1e99c3281cf66babbd2` |
| 申请站 1.0.1 ZIP | `592996324ae4b043d0e80a518b2897d4e86d743d977902ec18fa8813330638ff` |

当前目录分别是 `/opt/deuterium/releases/admission-email-57878e526653`、`/var/www/deuterium/releases/web-2.0.18-57878e526653/dist`、`/var/www/deuterium-admission/releases/admission-1.0.1-57878e526653/site`。

备份 `/var/backups/deuterium/admission-email-57878e526653`，数据库备份 SHA-256 `dcde7cfe859ff8bc9a342fba880f7019a98015f6fd9e4128212f6d58c0572c44`。030 迁移仅扩展邮件队列字段，旧迁移校验保持。回退优先前向修复；若回到不识别审核任务的旧 Go，必须先暂停邮件总开关，避免旧工作器误处理新任务。不得恢复整库覆盖新业务、已发送状态或已撤销资格。

[实施边界](../plans/admission-review-email.md) · [发布证据](artifacts/admission-review-email/verification.json)
