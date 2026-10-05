# 无效邮箱冻结候选与阿里云回执配置

2026-10-05 更新：SMTP 明确无效地址冻结与回调接口已随 [App 2.0.17 / Web 2.0.27](auth-flow-2.0.17-live.md) 上线。**真实阿里云 EventBridge 事件规则和服务器 Token 仍未配置，异步回执尚未接通。** 下文保留接入配置步骤；用户确认的冻结时长为 24 小时。

## 行为

阿里云 DM 明确拒绝地址后，按失败时间冻结该邮箱 24 小时。App、社区、官网、客服中心的注册、找回密码与修改密码发码请求统一返回 HTTP 400、`EMAIL_ADDRESS_INVALID`、`邮箱地址无效`。冻结请求不创建新邮件或消耗发码额度，已排队且尚未领取的验证码停止发送；正常账号登录不受影响。冻结不等于禁用账号。

SMTP 入信阶段只识别阿里云 DM 官方主机的 `559 Invalid rcptto` 和 `563 ... account-level bounce suppression list`；网络超时、认证、额度、垃圾邮件等错误不冻结。SMTP 自身的明确拒绝在新程序部署后即可记录，无需 EventBridge。

异步投递失败须完成下面的云端配置。回调按本系统 Message-ID 和收件人双重关联，必须是 DM 通道且发送次数大于零。只处理 `source=acs:dm`、`type=dm:Deliver:Fail`、`data.status=2`；相同邮件重复回执不延长时间，超过 24 小时的历史失败事件不重新冻结。未知邮箱、其他邮件、不匹配收件人不会写入。已在 SMTP 传输中的邮件无法撤回，不宣称能取消已在途投递。

## EventBridge 配置

1. 在服务运行环境设置独立随机 Token：`DEUTERIUM_DM_EVENT_TOKEN`，至少 32 字节；不要复用 SMTP 密码、账号 token 或把值写进源码。未设置时回调接口关闭。
2. 阿里云杭州地域的 EventBridge，为邮件推送的系统事件创建规则，筛选 `source=acs:dm`、`type=dm:Deliver:Fail`。邮件推送须已向该事件总线发布投递事件。
3. HTTP 目标使用 HTTPS：`https://chat.deuteriumix.com/api/v1/mail/aliyun/events`，POST Body 选择**完整事件**。目标高级选项的 Token 填步骤 1 同一值；EventBridge 会通过 `x-eventbridge-signature-token` 请求头传递。
4. 保留原始 CloudEvent `time` 和 `data.msg_id`、`data.rcpt`、`data.status`。已有 SMTP 发信头为 `<mail_...@deuterium-notifications>`，与 DM 的 `msg_id` 对应，不需要更换发信方式。
5. 发布时先部署数据库迁移与 Go，再开启回调规则；验证真实失败回执与已保存邮件能关联。只读核验已有用户数据，不能用无关真实邮箱做投递测试。

推荐筛选模式：

```json
{"source":["acs:dm"],"type":["dm:Deliver:Fail"]}
```

## 数据与回退

新增 `035_invalid_email_addresses.sql`，仅新建 `mail_invalid_addresses`，按邮箱和到期时间建索引。既有迁移不修改，账号、订单、工单、验证码及邮件原记录不清理。发布前备份数据库与当前程序。

到期后查询自然放行，保留每封邮件的失败证据用于去重，不因重复通知继续冻结。回退先停用 EventBridge 规则，再切回旧 Go；新表可保留，旧程序忽略它。回退旧程序会停止本地冻结检查，不应宣称冻结仍在生效；不恢复整库覆盖用户新增数据。

## 验证

- 普通 Go notify/config/httpapi/store 测试通过。
- 隔离数据库专项验证 SMTP 失败→冻结、统一公开错误、不增加队列/配额、事务内拦截、24 小时到期、重复事件不延长/复活冻结、回调鉴权、错误收件人/来源/历史时间忽略、晚到 SMTP 成功不覆盖失败、待发送验证码取消、找回/改密拦截及正常登录不受影响。
- SMTP 分类测试覆盖全局/个人无效地址库，以及第三方主机、类似域名、垃圾邮件、额度、网络故障等不得误冻结的场景。
- 各端现有 HTTP 400 处理会展示后端消息并恢复按钮；本轮未重跑真实四端 UI，也未宣称真实云回调已经收到。

官方依据：[投递事件字段](https://help.aliyun.com/zh/eventbridge/user-guide/events-of-the-direct-mail-type)、[目标 Token](https://help.aliyun.com/zh/eventbridge/user-guide/route-events-to-http-endpoints)、[SMTP 关联编号](https://help.aliyun.com/zh/direct-mail/asynchronous-notification-faqs/)、[559/563 错误码](https://help.aliyun.com/zh/direct-mail/user-guide/summary-of-common-email-return-codes-and-solutions)。
