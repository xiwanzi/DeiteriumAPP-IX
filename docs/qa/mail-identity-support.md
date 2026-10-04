# 邮件系统、独立游戏验证与邮件工单验收

2026-10-03（UTC+8）。工作区 `.worktrees/mail-identity-support`，分支 `xiwanzi/mail-identity-support`，基于主线 `05cc6dd`。本记录为本机候选，未推送、未合并、未部署。

## 实现结果

- 按用户选定方案二：游戏 ID、任意邮箱、邮件验证码及密码注册，随后补 QQ 并完成游戏验证码验证。账号主体先真实创建，服务器 UUID 等待 Core 验证后绑定；原账号 ID/playerRef 不变。
- 既有账号迁移默认已验证；身份验证不会授予、恢复或改动游戏白名单。未验证账号不能调用聊天、钱包及后台业务；账号仍可登录、退出、找回密码和使用未来 Help 邮件提交接口。
- App/Web 支持独立中文昵称，资料同时保留游戏 ID/QQ；昵称不用于身份及经济归属。
- 原 SMTP 栏目变为邮件系统，通道配置成为子项。原配置与授权码保留，自动通知切 DM，手动邮件和人工工单回复用企业邮箱。
- 工单完全由客服邮箱收件创建。未来 Help 仅将登录玩家的结构化内容经系统邮箱转交，Reply-To 是玩家邮箱；没有玩家在线工单对话。Help 邮件须通过一次性证明核对，不能靠 From/Reply-To 冒认玩家。
- 工单编号为 UTC+8 日期加四位随机数，共 12 位纯数字；建单与 DM 回执入队同事务。收到玩家后续邮件后归入原工单，原邮件和已读状态保留。
- 后台具备状态计数、问题类型颜色、队列/搜索/分类、指派与优先级、邮件回复、内部备注、快捷回复、附件下载、引用折叠和邮件投递状态；可发送并等待回复、继续处理或解决。
- 原完整管理员显示为超级管理员；新增 `support.manage` 管理员只有工单后台权限。

## 本次验证

| 范围 | 本次结果 |
| --- | --- |
| Go 全部普通测试、`go vet ./...`、Linux amd64 候选构建 | 通过 |
| 隔离 MariaDB 11.8.8 与 race | 邮箱注册、真实账号门禁、Core 绑定/Cookie 边界、验证码限次及并发消费、昵称不改变游戏身份通过 |
| 原审核邮件回归 | 原模板、收件隔离、私有审批备注隔离、暂停/取消、原队列重试及 SMTP 密文保留通过；测试通道按新规则配置 DM |
| 邮件工单 race | 直接收件、Help 先发邮件后建单、回复地址绑定、重复 UID/Message-ID 去重、自动回复抑制、线程及同邮箱主题编号归并、越权拒绝、内部备注不发邮件、失败重发保持原引用且不能跨工单通过 |
| 编号与附件 | 并发 10 封来信编号唯一；历史邮件接收边界；附件解析、加密保存、管理员下载及普通用户拒绝通过 |
| IMAP 本机 TLS 协议验证 | UID/UIDVALIDITY、EXAMINE、BODY.PEEK、收件时间及正文读取通过，没有 STORE/EXPUNGE |
| 邮件内容 | MIME/UTF-8、HTML 转文字、危险标签剔除、附件、回执转义及 Reply-To/In-Reply-To/References 头验证通过 |
| Web | 71 项测试、Vite 构建通过；有原有大包体积提示 |
| Android | `:ui-lab:assembleDebug :ui-lab:testDebugUnitTest` 通过，152 项测试零失败 |
| 浏览器 | 实际邮箱验证码注册后停留验证页、受限管理员仅工单导航、内部备注、发邮件并选择状态、浅深色、移动端及回执模板预览通过；无控制台错误 |
| 契约/差异 | OpenAPI YAML 解析及新账号路径/字段检查、`git diff --check` 通过 |

Go 曾进行完整集成回归，初次失败仅为被新注册契约替代的旧注册用例及尚未配置 DM 的旧邮件 fixture。随后将用例更新为完整“邮箱注册 → 游戏验证”流程，并对全部失败项及新业务执行专项 race 回归，通过；不将早期失败运行写成最终全量集成通过。

所有测试数据库均为回环地址的随机隔离库，SMTP/IMAP 使用本机合成内容和测试服务，不连接生产账号、发送真实客户邮件或修改真实白名单。

## 构建与截图

- 测试 APK：`android-app/ui-lab/build/outputs/apk/debug/ui-lab-debug.apk`。
- 包名 `com.deuterium.app.uilab`，版本仍为 **2.0.14 / 21400**；本轮未升发布版本，不能当成线上更新包。
- SHA-256：`483fd235689f8c09772adea9d822b219dd8c3664339e4cc59dc3c81c8ebe9a04`。
- Web：`web-app/dist`；浏览器使用隔离数据库与合成邮箱。
- 本机截图：`output/email-ticket-workspace.png`、`output/email-ticket-dark.png`、`output/email-ticket-mobile.png`、`output/ticket-receipt-preview.png`、`output/verification-web.png`、`output/verification-mobile.png`。截图与运行日志为本机产物，不提交数据库或运行目录。

## 尚未实测与发布条件

已确认杭州 DM `noreply@notify.deuteriumix.com`、客服 `support@deuteriumix.com`、Help 系统邮箱 `go@deuteriumix.com`。默认地址已设置，通道保持未启用。真实 SMTP/IMAP 授权码尚未配置，因此还未验证阿里云实际收发、送达收件箱及真实邮件客户端归并；本机 SMTP 接收测试不等于真实邮箱投递。

没有连接 Android 真机；本轮构建和单元测试不代表真机跳转手感验收。未来 Help 网页尚未创建，只提供后端邮件转交接口；接入该页面时需沿用受保护登录会话。

生产发布、版本升级、数据库迁移和最终邮箱联调须依后续授权执行。步骤和回退边界见[候选部署说明](../deployment/mail-identity-support-candidate.md)。
