# 工单邮件主题与数字字体修正

2026-10-04（UTC+8）**20:07:32** 已上线，仅替换 Go，App 2.0.15 / Web 2.0.24 保持。[PR #64](https://github.com/xiwanzi/DeiteriumAPP-IX/pull/64)，源码 `cf94741842ec1bf7322c904422cfbcbf6b9d69b2`，维护 `.worktrees/mail-identity-support` / `xiwanzi/mail-subject-digits`。

- 自动回执、主动建单、人工回复主题不再添加工单号，正文顶部编号保留；旧待发邮件在渲染时移除本单的 `[工单 编号]` 标记。In-Reply-To / References 与邮件关联数据保持，旧邮件的编号兼容查询仍保留。
- Git 历史确认：工单号原来使用 Consolas / Courier New，普通正文数字原来使用 Arial。本次恢复这些数字字形；中文 HarmonyOS Sans SC、六位验证码原 Consolas、品牌图和版式不变。字体回退仍取决于收件客户端。
- Go notify/store/httpapi 既有测试通过；主动建单、主题无编号、正文有编号、后续回复归并集成测试通过（4.341s）。线上预览主题为“已收到：游戏登录遇到问题”，模板声明工单号 Consolas、普通数字 Arial，旧 Novecento 声明已去掉。未另发真实邮件。

Go `/opt/deuterium/releases/mail-subject-digits-cf94741842ec`，21,577,888 字节，SHA-256 `958cb2961d8af3c059f3f3ff33f080e4fae3097f8010db656c61fa03074da931`。运行进程摘要与产物一致，ready、四节点在线，旧账号和业务 ID、配置与迁移保持；临时只读会话已撤销。

备份 `/var/backups/deuterium/mail-subject-digits-20261004-cf94741842ec`，数据库 gzip 20,850,472 字节，SHA-256 `d2d46169cfe29ff4887ad72fe127a6999b7aeb558387ae894ec430e519f2e93f`，解压校验通过。需回退时只切回 `/opt/deuterium/releases/outbound-support-0a6fea8d047c` 并重启 Go，不恢复数据库或配置。

[产物](artifacts/mail-subject-digits/manifest.json) · [备份](artifacts/mail-subject-digits/stage.json) · [切换](artifacts/mail-subject-digits/activated.json) · [线上模板验证](artifacts/mail-subject-digits/verified.json)
