# 阿里云无效邮箱临时冻结

用户确认：阿里云明确返回无效地址后，邮箱冻结 24 小时，之后验证码请求不发邮件，四端提示“邮箱地址无效”。到期自动允许再次尝试。保留现有账号、登录、资产及邮箱配置，不冻结账号。

实现边界：

- 阿里云 DM SMTP 明确的无效收件地址拒绝，以及经鉴权的 EventBridge `dm:Deliver:Fail` / `status=2` 投递事件，均可形成冻结证据。网络故障、垃圾邮件拒绝、认证/配额错误不作为无效地址。
- 回调须携带独立 Token，且 Message-ID 与收件邮箱匹配本系统实际发送过的 DM 邮件；不接受只给邮箱的冻结请求。按邮件去重，重复通知不延长冻结；过期历史通知不重新冻结。
- 冻结持久化到新增迁移，所有验证码用途、所有客户端共用；入队前及事务中检查，冻结请求不消耗发码额度。待发送的验证码也检查冻结。
- 第一封邮件异步确认无效后开始生效，当前界面不伪造同步投递结果；后续请求明确返回 `EMAIL_ADDRESS_INVALID` / “邮箱地址无效”。普通登录不受影响。
- 本机隔离库/模拟 SMTP/回调覆盖鉴权、收件人匹配、重复/迟到通知、24 小时到期、并发请求与非无效地址错误。真实阿里云回调必须完成云端规则配置后才能宣称接通。

依据：[DM 投递事件](https://help.aliyun.com/zh/eventbridge/user-guide/events-of-the-direct-mail-type)、[HTTP 目标 Token](https://help.aliyun.com/zh/eventbridge/user-guide/route-events-to-http-endpoints)、[SMTP Message-ID 关联](https://help.aliyun.com/zh/direct-mail/asynchronous-notification-faqs/)。
