# 登录注册修复与无效邮箱冻结上线

2026-10-05 **13:32（UTC+8）**，App **2.0.17（21700）**、社区 **Web 2.0.27**、官网/客服中心共享账号脚本及配套 Go 已上线。PR [#74](https://github.com/xiwanzi/DeiteriumAPP-IX/pull/74)，源码 `0248496b034aa7fdfd0977da5c57b514f30b8e5c`，合并 `9d55a6c972f86cc1d6b05a9f99a40136b8ef0d89`。

## 已生效

- 发码失败查询不占用成功发码额度，显示实际冷却等待；网络未知结果不会立即开放重发。
- 修复游戏 ID 找回已绑定邮箱账号的重复扣额度；App 找回密码弹窗关闭重开保留冷却，登录晚到的限流响应只归属原提交账号。
- App、社区、官网、客服中心统一验证码文案，删除发送排队提示、重复渠道解释，精简 App 找回/修改密码界面与社区表单间距。
- 阿里云 DM SMTP 明确返回 `559 Invalid rcptto` 或 `563` 个人无效地址库拒绝时，记录该邮箱 **24 小时**冻结。四端后续验证码请求返回 **“邮箱地址无效”**，不入队、不扣发码额度；待发送验证码停止，正常账号登录不受影响。
- App 在线更新已发布，包名及证书保持。20004、21300、21500、21600 均能取得 21700；21700 不重复提供更新。最低适用版本为 1，原 17 个发布条目保留。

## 明确未接通项

**阿里云 EventBridge 异步投递回执尚未配置。** 接收接口已部署，但服务器未设置 `DEUTERIUM_DM_EVENT_TOKEN`，公网验证返回 `MAIL_EVENTS_DISABLED`，没有把它描述为实际收到回执。SMTP 当场返回无效地址的冻结已可工作；“先接收邮件、后来投递失败”的结果需依[接入说明](invalid-email-events-candidate.md)配置云端规则后才进入冻结。

## 验证与数据

- [发布前验收](../qa/auth-release-217-20261005.md)：Go 普通测试/vet、隔离数据库认证与冻结专项、Web 71 项、共享会话 10 项、Android 152 项 JVM 测试、模拟器 8 项原生回归、三处网页隔离界面验证通过。
- 公网 APK 字节数及 SHA-256 与发布清单一致；官网、客服中心两份账号脚本分别与候选摘要一致；社区配置返回 2.0.27。
- 以临时 `example.invalid` 邮箱冻结记录验证四个真实入口均返回 `EMAIL_ADDRESS_INVALID` / “邮箱地址无效”，没有创建账号、验证码或邮件。临时冻结记录已删除。
- `deuterium.service` active / ready，Login、Amiya、Odyssey、MEK 四个 Core 节点在线。验证用短期管理员会话已撤销。
- 切换前后均为 **50 个账号、27 笔订单、5 个委托、12 个工单、1 条注销记录**；配置摘要不变。只新增 035 冻结表，原迁移摘要全部保持，未更换游戏插件。
- App 已完成模拟器覆盖安装及原生交互测试，不冒充用户真机触感验收。此次没有另发账号更新通知，用户可在 App 检查更新。

## 程序与备份

| 对象 | 当前目录 |
| --- | --- |
| Go | `/opt/deuterium/releases/auth-flow-0248496b034a` |
| 社区 | `/var/www/deuterium/releases/web-2.0.27-0248496b034a/dist` |
| 官网 | `/var/www/deuterium-official/releases/official-auth-0248496b034a` |
| 客服中心 | `/var/www/deuterium-help/releases/help-auth-0248496b034a` |

备份：`/var/backups/deuterium/auth-flow-20261005-0248496b034a`，含停服后的单事务数据库备份、旧程序、配置、原更新清单和迁移摘要。旧目录均保留；如需回退，可切回程序/静态目录及原清单，不用整库还原覆盖用户新数据。冻结表可保留，旧 Go 不执行本地冻结检查。

APK **8,908,306** 字节，SHA-256 `893a664ead271636140af6aff27823394f65c05a43272be26d15707b331a6751`。签名 SHA-256 `04a213066a78a968c2a1328a153a0967f99d117e0a826285e3f4af705db6f2f3`。

官网维护源 `C:/Mod/DeuteriumIX/work/review/d9-endfield/`、Help 维护源 `.worktrees/support-center-preview/support-center/` 的两份账号脚本，与本次已合入主线的 `artifacts/v217/` 源码快照一致。

[构建清单](artifacts/v217/manifest.json) · [部署证据](artifacts/v217/deployment.json) · [公网验证](artifacts/v217/verification.json)
