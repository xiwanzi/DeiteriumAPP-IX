# 审核邮件配色保护与小图标上线

2026-09-13 **10:47:28（UTC+8）**，通过与拒绝两套正式邮件已同步用户确认的深色模式配色保护和小尺寸内嵌徽章。[PR #37](https://github.com/xiwanzi/DeiteriumAPP-IX/pull/37) 已合并，运行代码对应 `a79eb664d28d49d8a7762518fcb32dec2357bafe`，构建提交 `f87d15598a88b685d89d7be85e9f0f23c1364d3f`。

模板增加 `only light` 声明、明确文字填色与背景保护，保留已确认的字体、正文、动态游戏 ID 和拒绝原因。QQ 等客户端仍拥有最终渲染控制，不宣称所有版本都绝对锁色。

徽章保持原设计和透明背景，由 1254×1254、1,819,447 字节改为 **132×132、32,512 字节**，邮件显示仍为 44×44。继续使用 CID 内嵌附件，未切换到外链；后台预览也使用同一小图。此前用户在 QQ 与 Gmail 收到的小图标测试邮件约 71 KB。

本次仅更新呈现和资源，逻辑模板版本与审批/重试接口不变；后续发送均使用新呈现。Web 保持 2.0.18，App 保持 2.0.13，游戏插件未更换。SMTP 设置仍为版本 9，审核邮件开关保持开启。

通知模块 race 测试与 vet 通过，验证 MIME 图片完整、转义、邮件体积上限与配色保护。上线后只读核对后台两种模板均包含保护样式、132×132 的 32,512 字节图片；运行进程二进制哈希一致，网关在线。未额外发送测试邮件。账号、交易、白名单状态、SMTP 配置/密文、启动器与 App 发布清单均保留。

新 Go：`/opt/deuterium/releases/mail-appearance-f87d15598a88/deuterium`，19,873,952 字节，SHA-256 `47e9c79dc4e9fa7b6c7605ed53f4d1eb4a1a4a67e99f543524a7a6077d67b23c`。备份 `/var/backups/deuterium/mail-appearance-f87d15598a88`，数据库备份 SHA-256 `fe438812444eff720416df3c4a2e1366fc5301a8e5390984c9e28819d56742a6`。旧 Go `admission-email-57878e526653` 保留，若有必要只回退二进制，不恢复整库或旧白名单资格。

[验证记录](artifacts/mail-appearance/verification.json) · [审核邮件接入记录](admission-review-email-live.md)
