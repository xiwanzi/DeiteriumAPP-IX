# 邮件系统、游戏验证与工单接口

2026-10-03 本机实现。地址前缀 `/api/v1`；部署状态以 QA/候选记录为准。

## 账号

| 方法与路径 | 输入 / 输出 | 权限与行为 |
| --- | --- | --- |
| POST `/account/registration-code` | `gameId,email` → `verificationToken,expiresAt,resendAfterSeconds,delivery=EMAIL` | 无需密码；DM 已配置启用时发信；兼容忽略旧 password 字段 |
| POST `/account/register` | `verificationToken,code,password` → Bearer 与 user | 仅 App；禁止 Origin；创建真实未验证账号 |
| POST `/web/register` | 同上 → HttpOnly Cookie、csrfToken 与 user | 精确来源检查，不返回 Bearer |
| POST `/account/game-verification-code` | `qq` → verificationToken 等 | 当前未验证会话；游戏 ID 从当前账号读取，Core 解析 UUID |
| POST `/account/game-verification` | `verificationToken,code` → user | 凭证须属于当前账号；消费与绑定同事务，原账号 ID/playerRef 不变 |
| POST `/account/password-reset-code` | `account` | 有邮箱账号发邮箱验证码，旧无邮箱账号仍使用游戏验证 |
| POST `/account/password-reset` | `verificationToken,code,newPassword` | 对应类型验证，修改密码后撤销已有会话 |
| POST `/account/password-change-code` | 当前会话，无需提交账号标识 | 邮件或旧游戏验证码；用途固定 password_change |
| POST `/account/password-change` | `verificationToken,code,newPassword` | 必须匹配当前会话账号与用途，成功撤销全部会话 |
| PATCH `/account/me/profile` | 原版本/请求键，新增可选 `displayName` | 昵称 1–24 字符、支持中文、禁止换行；不改变游戏 ID |

user 增加 `email`（本人可见）、`gameVerified`、`displayName`；`identityStatus` 为 `bound` 或 `unverified`。公开资料新增 displayName，不暴露邮箱。

未验证时除会话查询、退出、游戏验证、密码更改和未来 Help 邮件提交接口外，鉴权接口返回 HTTP 403 / `GAME_VERIFICATION_REQUIRED`，客户端保留账号会话并显示验证页，不当作密码错误。未验证账号不进入玩家目录、私聊目标和钱包收款人列表。现有账号迁移默认已验证，账号停用/注销仍优先拒绝。

邮箱验证码 10 分钟、每邮箱 60 秒冷却、每日 20 次；错误 5 次失效，消费同事务。公开发码还沿用 IP 限额。OTP 只存散列，邮件正文加密入队；消费、过期或被新验证码替换后取消尚未投递的旧验证码，发送后清除队列中的验证码密文。请求不返回明文验证码。

2026-10-04 验证码模板分别支持注册、找回密码和更改密码，后台通过 `/admin/mail/templates/verification/{register|password_reset|password_change}` 预览。带版本的用途、游戏 ID 和验证码参数加密入队，发送前生成 HTML、纯文本及 CID Logo。帮助中心仅预留 `https://help.deuteriumix.com` 链接。

## 邮件后台

仅超级管理员 `platform.admin`：

| 方法与路径 | 行为 |
| --- | --- |
| GET `/admin/mail` | DM、ENTERPRISE、SYSTEM 通道（仅 passwordConfigured）；通知配置、新邮件及原队列投递记录 |
| PUT `/admin/mail/channels/{DM\|ENTERPRISE\|SYSTEM}` | `clientRequestId,expectedVersion,settings,password?`；缺省 password 保留密文 |
| PUT `/admin/mail/notifications` | `clientRequestId,expectedVersion,admissionReviewEnabled,recipients[]`；保留原 SMTP 其他字段和授权码 |
| POST `/admin/mail/send` | `clientRequestId,recipient,subject,body,channel`；手工邮件界面使用 ENTERPRISE |
| 原 `/admin/email-settings` 及 templates/test | 保留原配置、模板和投递数据；自动通知实际由 DM 发信 |

所有 SMTP 使用 TLS 或 STARTTLS，复用现有服务器邮件加密密钥。客服收件建单、编号分配和回执入队原子提交；发送失败按队列恢复。`SENT` 表示 SMTP 接收，不能声称进入收件箱。新队列自动重试最多 8 次后 FAILED；验证码超时取消。SMTP 接收成功但响应丢失仍可能重复，重试保留 Message-ID，不声称邮件具备严格恰好一次保证。

## 客服收件与未来 Help

工单以真实客服邮箱收到邮件为入口，没有玩家工单列表或在线回复接口。

- PUT `/admin/mail/inbox`：仅超级管理员，`clientRequestId,expectedVersion,settings,password?`。settings 为 `enabled,host,port,username,address,folder,importExisting`；TLS 强制开启，密码不回传。GET `/admin/mail` 包含脱敏收件设置及同步状态。
- GET `/support/contact`：公开客服邮箱地址，供 App/Web 打开邮件应用使用。
- POST `/support/submissions`：未来 Help 页的已登录接口，`clientRequestId,subject,category,body`；只从 SYSTEM 内部企业邮箱向客服邮箱发一封邮件，Reply-To 从当前账号读取。返回 `submissionId,eventId,status=QUEUED`，不返回或预创建工单编号。
- SYSTEM、客服 ENTERPRISE SMTP 与收件 IMAP 分别配置。转交邮件携带随机一次性证明，服务端保存散列，发送标记加密入队。收件时检查 From、Reply-To 与证明，后台显示真实玩家邮箱；无效证明不会冒认已登录玩家。
- 直接来信可以来自没有平台账号的人；关联到已有邮箱账号仅作为支持人员上下文，不代替登录或经济操作身份验证。
- IMAP 每 10 秒尝试同步；单轮最多 25 封、60 秒超时，持久化租约防止多进程重复处理。使用 EXAMINE/只读与 BODY.PEEK，不删除原邮件、不更改已读标记。
- 默认从配置启用时间接收新邮件（按 IMAP INTERNALDATE），避免首次同步前到达的邮件被跳过。明确勾选 importExisting 才处理历史来信；UIDVALIDITY 变化重新扫描并按 Message-ID 去重。
- 单封原始邮件上限 8 MiB，最多 10 个附件；解析失败/超限显示 NEEDS_REVIEW，原邮件仍留在客服邮箱。HTML 转为可读文本，附件加密保存，下载始终检查工单权限并强制 attachment/nosniff。
- 自动回复、退信和系统自身邮件不创建工单，避免自动回执循环。

## 邮件工单后台

基址 `/admin/support/tickets`，要求 `support.manage` 或 `platform.admin`。

- 编号是 **12 位纯数字**：UTC+8 创建日期 `YYYYMMDD` 加随机四位（含前导零）。同一天从尚未使用的 0000–9999 中均匀随机选取并加数据库日锁；每日最多 10000 件，编号空间用完时保留来信等待处理，不改用其他编号格式。
- GET 列表：`status,queue,category,q,offset`，每页 50 条与 hasMore；queue 支持 mine/unassigned；可搜索主题、工单编号、邮箱或游戏 ID。
- GET `/{ticketId}`：邮件往来、内部备注、附件、申请人资料和邮件投递状态。工单包含 `requesterEmail,requesterName,source=EMAIL|HELP`。
- POST `/{ticketId}`：`clientRequestId,expectedVersion,body?,visibility?,status?,priority?,assigneeId?,category?`。PUBLIC 是发邮件，INTERNAL 是仅后台备注；发送与更新状态可一次提交。
- 状态：OPEN（待审查）、IN_PROGRESS、WAITING_CUSTOMER、RESOLVED、CLOSED。新来信重新打开为 OPEN；人工回复默认等待回复，也可同步选择继续处理或已解决。
- 类型：ACCOUNT、GAME、ORDER、ACCESS、OTHER。Help 使用表单类型，直接来信由主题初分，可由支持人员更正。优先级 LOW/NORMAL/HIGH/URGENT。
- GET `/{ticketId}/attachments/{attachmentId}`：仅工单工作人员可下载该工单附件。
- POST `/{ticketId}/mail/{eventId}/retry`：工单人员可对该工单已 FAILED 的业务邮件重新入队，保留原 Message-ID；全局 `/admin/mail/events/{eventId}/retry` 仅超级管理员。验证码禁止重发旧任务，应重新获取。
- GET `/admin/support/tools`：负责人、快捷回复、状态统计和脱敏收件运行状态。
- PUT `/admin/support/macros/{macroId}`：`clientRequestId,title,body,version`，version=0 新建。
- GET `/admin/mail/templates/support-received`：超级管理员预览 DM 收件回执模板。回执包括主题、摘要及工单编号，不显示「待审查」；Reply-To 为客服邮箱，玩家直接回复即可接续。

归并优先使用 In-Reply-To / References 中的邮件编号，且要求发信回复邮箱与工单玩家邮箱一致；缺少引用时，只有主题中明确的 `[工单 YYYYMMDDNNNN]` 与相同邮箱可以接续。不同玩家不能凭工单号或伪造引用并入别人的邮件记录。收件与出件保存稳定 Message-ID，重复拉取或 Help 重发不重复建单和自动回执。

角色：原 `platform.admin` 显示超级管理员；新增 `support.manage` 显示管理员，仅访问工单管理。账号管理内部动作 grant-admin/revoke-admin 保留为超级管理员语义；新增 grant-support/revoke-support 用于工单管理员。版本冲突返回 `STATE_VERSION_CONFLICT`，草稿保留；请求键重放不重复发信。

所有邮件遵循[邮件品牌与字体规范](mail-visual-standard.md)：正文使用官网鸿蒙字体，数字使用官网 Novecento，品牌字标与徽章内嵌，保留可复制文字和明确的客户端回退。
