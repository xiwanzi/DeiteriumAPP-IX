# 工单工作区效率优化上线（Web 2.0.22）

2026-10-04（UTC+8），**13:46:36** 完成 Web 2.0.22 与配套 Go 切换。[工单工作台](https://chat.deuteriumix.com/admin?section=support)现在默认进入紧凑、全宽的处理界面。App 仍为 **2.0.15 / 21500**。

## 本轮结果

- 默认将全站导航收为 64px 图标栏，可随时展开；取消工单页的装饰标题、统计卡片、最大宽度限制和站点页脚。
- 1440×900 实测工作区 1376×806，占视口约 85.6%；正文可视高度从旧版约 181px 增至 493px。发送按钮始终在视口内，列表/对话分别滚动。
- 状态数量与筛选合并；输入即搜索。负责人、状态、由我处理、解决、上一条/下一条放在对话顶部；资料按需打开，手机为抽屉。
- 各工单分别保留邮件/内部备注草稿；发送按钮明确标示目标状态，成功后可自动进入下一条。失败、并发冲突和响应丢失不丢草稿、不提前跳转、不重复发送。
- 修复中文查询同时搜索 ASCII 工单号/邮箱时的 MariaDB 字符集冲突；只改查询，不改变数据库结构。
- 邮件正文模板、署名、HTTPS 品牌图、Gmail 折叠约定、SMTP/IMAP 密码及通道设置全部保留。

## 源码与产物

[PR #58](https://github.com/xiwanzi/DeiteriumAPP-IX/pull/58)，产品提交 `8d4ed9870592`，合并源码 **`939d0b780f77fd967e8ec6f8e95354f100fc0538`**。维护工作区 `C:/DeiteriumAPP-IX/.worktrees/mail-identity-support`，分支 `xiwanzi/support-workspace-efficiency`。

| 产物 | 字节数 | SHA-256 |
| --- | ---: | --- |
| Go Linux amd64 | 21,549,216 | `a3c8baf6b59d3be3f355c85025ac9b63adadd15e80fdbfd44d4a0a87cd1975bf` |
| Web 2.0.22 ZIP | 33,738,605 | `3a14780df3a4645258f507a928f9e76a50781996d8c9d7f1ddd9a3baff3fb87e` |

Go 使用 Go 1.27.1、CGO=0、`-trimpath -ldflags='-s -w'`，干净合并源码构建，`vcs.modified=false`。Web 源码与合并版本一致；生产 `web-config.json` 从原配置仅更新版本，保留连接模式及其余字段。

- Go：`/opt/deuterium/releases/support-workspace-939d0b780f77`。
- Web：`/var/www/deuterium/releases/web-2.0.22-939d0b780f77/dist`。
- 本机产物：`output/support-workspace-release/`。

## 验证与数据

[本机验收](../qa/support-workspace-efficiency.md)：Web 71 项测试与构建，Go store/httpapi 普通测试及 vet，搜索和邮件工单专项 race；浏览器处理流程、故障恢复、权限、8 种视口及浅深色通过。

线上登录工作台只读检查：1440×900 工作区与本机测量一致；390×844 无横/纵溢出，发送按钮在视口内。中文搜索、中文空结果、编号查询均成功；无页面脚本异常。公网 JS/CSS 摘要与构建一致，Web 版本为 2.0.22。临时验收会话已撤销，没有操作真实工单状态或给客户发送测试邮件。

切换前后既有账号、业务资源、工单和消息 ID 集合保留：**8 个账号、3 张工单、27 笔订单、5 个委托、0 注销记录**。三条邮件通道继续启用，收件无错误，Login/Amiya/Odyssey/MEK 四节点在线。28 个迁移的摘要未变，未执行迁移。App 更新清单、启动器与游戏插件保持。

首次切换预检发现并行工作更新了 `config.json` 的 `additionalPublicOrigins`，在停止服务之前中止。重新读取并额外备份该最新配置后继续，未覆盖域名设置；环境文件、服务配置和邮件设置保持原摘要。

## 备份与回退

备份目录 `/var/backups/deuterium/support-workspace-20261004-939d0b780f77`，目录 0700、敏感文件 0600。数据库一致性备份 20,592,247 字节，SHA-256 `14afdfa55a5438830fd466264ed45b4c3262f3e40e6dd829ae2e6b5f5d2734e0`，解压完整性通过。包括最新配置副本 `config-before-switch.json` 和切换前 ID 状态；未导入旧备份。

若需回退，只将 Go 指向 `/opt/deuterium/releases/mail-fix-b0b6686cf1ad`、Web 指向 `/var/www/deuterium/releases/web-2.0.21-6d5a2fdde8f1/dist` 后重启 Go；同一结构下保留全部当前数据及配置，不整库恢复，不恢复已清理账号。回退后会恢复旧工作区与中文搜索缺陷，优先前向修复。

[产物](artifacts/support-workspace/manifest.json) · [上传与备份](artifacts/support-workspace/stage.json) · [切换](artifacts/support-workspace/activated.json) · [公网文件](artifacts/support-workspace/public-verification.json) · [线上浏览器](artifacts/support-workspace/browser-verification.json)
