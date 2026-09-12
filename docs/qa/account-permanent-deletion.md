# 永久注销账号验收

2026-09-12。实现提交 `038718b068fb8a6a4074a60d4bd0cdb3fbd567c0`，分支 `xiwanzi/account-erasure`，工作区 `.worktrees/account-erasure`。基于 App 2.0.12 / Web 2.0.10 配套源码。

**本地候选，尚未推送、合并或部署；没有注销真实账号，`luoyinwuchen` 未在本轮被操作。** App 版本仍是 2.0.12（21200），本机 Debug APK 仅用于验证，不是新的线上发布。

## 行为

管理工作台“账号与权限”新增红色“永久注销”。弹窗说明清理范围，显示当前管理员账号名，只接受当前管理员密码；空密码不能提交，失败/关闭清空输入，提交期间不能关闭或重复执行。请求回执丢失时，重新输入密码用同一请求键确认原结果。

后端注销终态不可恢复，移除登录绑定和会话、主页、联系人/关注、相关私聊、公开作者消息、市场商品及当前资料图片绑定；App/Web 同步清除引用和缓存，旧异步请求不能重新添加资料。历史订单、转账和审计保留并匿名化当事人；同一游戏身份的新注册使用新账号引用。

有未完成订单/委托、退款/介入、待处理或未知转账时拒绝注销。持有商店所有权时先处理商店归属。不能注销当前管理员或移除最后一个有效管理员。

## 验证记录

| 项目 | 结果 |
| --- | --- |
| Go 竞态/数据库全量集成 | `go test -race -tags integration ./... -timeout 8m` 通过，见[输出](artifacts/account-erasure/go-full.txt) |
| 收尾转账/审计边界 | 全量检查后补齐历史转账回执、新注册引用和注销审计筛选，针对 AccountDeletion、GameOnlyTransfer、Wallet、Ledger、Transfer、AdminAudit 重新执行竞态测试，通过，见[输出](artifacts/account-erasure/go-boundaries.txt) |
| 密码与权限 | 当前管理员密码通过；目标密码/错误密码拒绝；失败限次、Web Origin/CSRF、本人保护、凭证变化和会话撤销均覆盖 |
| 终态与并发 | 重试只注销一次；两位管理员互相注销时保留一位；旧账号不能通过解封恢复，后续旧身份写入拒绝 |
| 数据清理 | 主页/旧会话/市场商品返回不可见，关注及私聊移除，消息转发/引用不泄露原文；其他账号和会话保留 |
| 财务保留 | 完成订单匿名显示，原快照 SHA 不改；处理中/UNKNOWN 转账阻止注销，完成后的原请求重试只返回旧回执；新付款拒绝已注销收款人 |
| 图片边界 | 资料绑定和未绑定私有图片被移除，仍被他人订单快照使用的图片保留 |
| 再注册 | UUID/QQ 可验证后注册，userId/playerRef 不复用，旧私聊和旧订单归属不继承；注册前的游戏转账回执继续可读 |
| Go 静态检查 | `go vet ./...` 通过 |
| Web | 68 项测试通过，生产构建通过；有约 501KB 主包的 Vite 大小提示，没有编译错误 |
| 管理台浏览器 | 桌面、390px 深色布局、空密码、错误密码清空、未知回执同键重试、单次执行、保留其他用户、提交期间禁止关闭均通过 |
| Android | `assembleDebug`、`assembleDebugAndroidTest`、152 项 JVM 测试通过；lint 0 errors、24 warnings |
| Android 模拟器 | `accountErasure` 三组检查 PASS：联系人/关注/商品/消息/草稿清理；延迟主页不复活、注销标记持久化；已打开主页/会话及旧通知链接不可用 |

全量回归首次发现了新增状态检查过早固定数据库快照，导致并发上传/消息重试冲突。账户相关事务改为 READ COMMITTED，继续保持现有资源锁和版本检查，再次全量回归通过。最终钱包/审计收尾改动按相关边界定向复验，没有把先前全量运行描述为这些收尾改动之后又跑了一遍全量。

所有数据测试使用回环地址的随机隔离 MariaDB 库。浏览器验收使用 `web-app/qa/account-erasure.html` 的虚构用户和内存响应；原生验收使用隔离偏好及 HTTP 拦截，不连接真实删除接口。模拟器证据不等于真机手感验收。

## 界面

[管理台密码确认](artifacts/account-erasure/web-confirmation.png) · [手机深色](artifacts/account-erasure/web-mobile-dark.png) · [注销完成](artifacts/account-erasure/web-completed.png)

[App 其余联系人保留](artifacts/account-erasure/app-contacts.png) · [旧主页](artifacts/account-erasure/app-profile.png) · [旧会话](artifacts/account-erasure/app-conversation.png) · [旧链接不可生成空白资料](artifacts/account-erasure/app-unresolved-profile.png)

[机器摘要](artifacts/account-erasure/summary.json) · [Web 测试](artifacts/account-erasure/web-tests.txt) · [接口契约](../contracts/account-permanent-deletion.md) · [实施边界](../plans/account-permanent-deletion.md)

## 后续发布

发布需要新增迁移 `027_account_deletion.sql`，先备份数据库，再部署 Go/Web，最后发布具有注销同步能力的 App。迁移本身不删除用户；管理员须在真实工作台输入自己的密码执行注销。

离线或未更新客户端无法立即清理既有缓存；外部截图、自行复制的非结构化内容及受限备份不能被此功能远程收回。回退不能重新激活已注销账号，也不能覆盖掉注销后产生的业务数据。
