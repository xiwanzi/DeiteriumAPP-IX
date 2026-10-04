# 玩家支持名称统一上线

2026-10-04（UTC+8），**14:09:10** 上线 Web **2.0.23** 与配套 Go。[PR #60](https://github.com/xiwanzi/DeiteriumAPP-IX/pull/60)，合并源码 `77af63a1b2907fe1567d0cef2f4c2cdd74b2fe1f`。App 保持 2.0.15（21500）。维护 `.worktrees/mail-identity-support` / `xiwanzi/player-support-name`。

自动工单回执原先显示“Deuterium IX 客服”，人工回复显示“Deuterium IX 玩家支持”。本轮统一为 **Deuterium IX 玩家支持**，两个渲染器共用名称常量；邮件纯文本联系标签、Web 邮件管理、工单消息标识和受限管理导航一并统一。邮箱地址、通知用途、个人署名、版式与收发协议不变；已发送历史邮件不重写。名称规范已加入[邮件视觉约定](../contracts/mail-visual-standard.md)。

既有 Go notify/httpapi/store 测试、Web 71 项测试及生产构建通过。直接调用两个实际渲染器核对发件人均为新名称、旧联系标签不存在；公网 Web 版本与 JS/CSS 摘要一致，旧“客服邮箱/客户支持”页面文案不存在。没有另发真实测试邮件、修改玩家历史工单或重新配置邮箱。

Go `/opt/deuterium/releases/player-support-name-77af63a1b290`；Web `/var/www/deuterium/releases/web-2.0.23-77af63a1b290/dist`。Go SHA-256 `bf59690e8b5d3253a1101988c93515d510aaae930f4b2cb3b5074e1e5ec9392a`，干净合并源码构建。切换后服务 ready，原账号、工单、消息和交易 ID 保留：8 账号、4 工单、27 订单、5 委托、0 注销记录；4 张工单含本轮开始前的正常新增，不是验收新建。28 个迁移未变，App 清单、邮件设置及启动器配置摘要保持。

备份 `/var/backups/deuterium/player-support-name-20261004-77af63a1b290`，数据库 gzip 20,608,213 字节，SHA-256 `eb59c7465683e9f40caff6e7b5d8b362d5af0aae48bc0adb28b4a64c1ad6b0eb`，解压校验通过。回退仅切回 Go `/opt/deuterium/releases/support-workspace-939d0b780f77` 和 Web `/var/www/deuterium/releases/web-2.0.22-939d0b780f77/dist`；不恢复数据库或旧账号，不回写邮件配置。

[产物](artifacts/player-support-name/manifest.json) · [备份](artifacts/player-support-name/stage.json) · [切换](artifacts/player-support-name/activated.json) · [名称渲染](artifacts/player-support-name/rendered-names.json) · [公网验证](artifacts/player-support-name/public-verification.json)
