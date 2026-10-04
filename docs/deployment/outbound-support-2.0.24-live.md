# 主动建单与管理员游戏 ID 修改上线

2026-10-04（UTC+8），**19:56:12** 完成 Web **2.0.24** 与配套 Go 切换。App 保持 **2.0.15（21500）**。

[工单工作台](https://chat.deuteriumix.com/admin?section=support) · [账号与权限](https://chat.deuteriumix.com/admin?section=accounts)

## 已上线能力

- 工单工作台新增“新建工单”：工作人员填写邮箱、主题、类型及正文后创建并发送。首封沿用“Deuterium IX 玩家支持”人工邮件模板，主题及正文顶部带工单编号；**不发送“工单已收到”自动回执**。
- 默认由创建人负责、状态为等玩家回复。玩家来信接续原工单，仍不补发回执；尚未收到来信前也能继续人工回复。创建、消息、发信任务、线程关联及审计使用同一幂等事务。
- “账号与权限”新增超级管理员专用的“修改游戏 ID”。未验证账号可纠正填写错误；已验证账号的新名称须由 Core 确认仍属于原游戏身份。不会改写 Minecraft 服务器用户名或迁移到其他 UUID。
- 同步更新游戏 ID 登录别名，保留原内部身份、资产、QQ、邮箱、昵称字段与权限；失效旧游戏验证码并撤销目标账号旧会话，用户重新登录取得新信息。新名称占用、版本冲突和未知结果均有恢复流程。

## 源码与产物

[PR #62](https://github.com/xiwanzi/DeiteriumAPP-IX/pull/62)，产品提交 `347247e586ee`，合并源码 **`0a6fea8d047c5f40a01fdfea0170c171b3a141ce`**。维护工作区 `C:/DeiteriumAPP-IX/.worktrees/mail-identity-support`，分支 `xiwanzi/outbound-support-ticket`。

| 产物 | 字节数 | SHA-256 |
| --- | ---: | --- |
| Go Linux amd64 | 21,577,888 | `e4330a3811c04c86fb46a6a4e04e1eea7cdf0a2809a8f076c70ccefa9dda8fa1` |
| Web 2.0.24 ZIP | 33,740,173 | `8b7d3b93846bbaad0ead14935ddc30f1d21ac90bf1aba363097462c8e3939db1` |

Go 从干净合并源码构建，Go 1.27.1、CGO=0、`-trimpath -ldflags='-s -w'`，`vcs.modified=false`。Web 源码与合并版本一致，线上公共配置保留原字段，仅更新版本。

- Go：`/opt/deuterium/releases/outbound-support-0a6fea8d047c`。
- Web：`/var/www/deuterium/releases/web-2.0.24-0a6fea8d047c/dist`。
- 本机发布包、执行记录：`output/outbound-support-release/`。

## 验证与保留数据

[本机验收](../qa/outbound-support-ticket.md)包括：Go 普通测试与 vet、6 项定向 race、网页 Origin/CSRF 拒绝、Web 71 项测试/构建、实际渲染首封编号邮件且无回执、回信归并、权限/并发/版本/资产身份/旧验证码，以及两个表单提交成功但响应丢失后的幂等恢复。桌面、手机及浅深色通过。

上线后只读打开真实工作台的新建弹窗和超级管理员改名弹窗，入口、字段、空内容/未改名禁用提交及关闭正常，页面无脚本错误。公网版本和 JS/CSS 摘要与构建一致。**未为验收创建真实工单、向真实玩家发邮件或修改真实游戏 ID。** 临时验收会话已撤销。

本轮预检时已有 **40 个账号、6 张工单、27 笔订单、5 个委托、1 条注销记录**，切换后原账号、业务、工单和消息 ID 集合保留。较早记录中的 8 账号/4 工单/0 注销不是本轮基线，不以其覆盖当前正常业务。邮件配置、App 更新清单及启动器配置摘要保持；28 个迁移未变，未执行数据库迁移；三个邮件通道启用、收件无错误，四个 Core 节点在线。

## 备份与回退

备份目录 `/var/backups/deuterium/outbound-support-20261004-0a6fea8d047c`（目录 0700，敏感文件 0600）。数据库一致性 gzip 20,834,957 字节，SHA-256 `0d967e6f03e75bb7965dd2acbca02df72da3aace2f316585964562cdfb63e23b`，解压校验通过；配置及切换前身份/业务 ID 状态一并保留。

前一版 Go 为 `/opt/deuterium/releases/player-support-name-77af63a1b290`，Web 为 `/var/www/deuterium/releases/web-2.0.23-77af63a1b290/dist`。回退只切换程序/静态目录并重启 Go，不恢复数据库、旧账号或旧配置。若已有主动工单，旧 Go 在玩家首次回信前不能继续人工回复，优先前向修复；历史工单、已改名账号和新邮件关联数据必须保留。

[接口契约](../contracts/openapi-support-admin-actions.yaml) · [产物](artifacts/outbound-support/manifest.json) · [备份](artifacts/outbound-support/stage.json) · [切换](artifacts/outbound-support/activated.json) · [公网](artifacts/outbound-support/public-verification.json) · [线上浏览器](artifacts/outbound-support/browser-verification.json)
