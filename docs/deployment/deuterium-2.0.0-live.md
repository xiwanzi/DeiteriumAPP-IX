# Deuterium 2.0.0 测试部署记录

更新：2026-09-08 14:37。应用、网页与 Go 后端已发布，Core / Sync / Mail 已安装四服并通过三个常驻节点运行验收。用户实测发现的转账回执、图片显示、客服菜单、分类重叠与版本标问题已修复并发布 20004。

## 访问与安装

- 网页与 API：<https://47.103.99.34>，标准 HTTPS 443。
- APK：<https://47.103.99.34/downloads/Deuterium-2.0.0-20004-test.apk>。
- App 版本 `2.0.0 (20004)`，包名 `com.deuterium.app.uilab`，Android 8.0+；延用 09 的签名，支持覆盖安装。首次真实登录使用原注册账号；旧演示身份和模拟资产不会成为真实数据。
- 域名仍按本轮用户说明等待备案，本次使用已验证的 IP 证书。续期定时器与 Nginx 重载已配置并完成 dry-run。

| 交付对象 | 已发布版本或目录 | SHA-256 |
| --- | --- | --- |
| Android APK | 20004，22,684,005 bytes | `bb48ece21019d8ba25e554e2e6b13f94876545bf9019de5874afc127a2c5f7a3` |
| APK 签名证书 | 原 Android Debug 测试签名 | `04a213066a78a968c2a1328a153a0967f99d117e0a826285e3f4af705db6f2f3` |
| Go 后端 | `/opt/deuterium/releases/wallet-proof-hotfix-20260908`，源码 `8744f5b` | `f9c87f969e01adde537933095986b0aa13786c04be12975d15a7b22c2028af14` |
| 网页静态包 | `/var/www/deuterium/releases/web-2.0.0-4de1ee7/dist` | 归档 `c92bc90edfab96a76138bf416c1c82fc65c9c00111d97a9b8014a9b60dddbb76` |

已完整下载公网 20004 并核对大小/摘要；旧 20003 检查更新得到 20004，发布清单最低版本为 1，与 App 校验规则一致。APK 与网页归档按本轮服务、节点及测试会话凭据做精确字节扫描，未发现这些凭据；源码与交付清单也不包含私钥或 QA 会话文件。

## 已部署服务与验证

| 能力 | 实际状态与验证边界 |
| --- | --- |
| 账号 | 70 个原账号、139 条有效别名已迁移，原 ID/UUID/密码哈希保留；唯一非数字旧 QQ 保留展示但不创建无效 QQ 别名。App Bearer、网页 Cookie/CSRF、授权与会话撤销已测试；没有替用户重置密码。 |
| 钱包转账 | 真实 xiwanzi → luoyinwuchen1 的 1.00 CREDIT 已完成；原键重放无重复，同键改额 409，App/Web/实服账本一致。两端均从已安装/已发布产物读取到原交易。 |
| DIMA / DaoYu | 已由 Amiya 控制台创建为 XConomy 系统专户，初始余额各 0.00、hidden=1；不创建 App/QQ/密码身份。官方收入进入 DIMA，预付进入 DaoYu，退款原路返回。 |
| 聊天 | App / Web 同一公共消息实时互通，三个常驻游戏节点 ACK，重复请求历史仍一条；用户明确授权的测试内容保留真实记录。私聊、引用、转发、关注及接收者隔离已用隔离数据库双账号验收，相关 API 已部署。 |
| 商城 / 市场 / 委托 | 目录、购物袋、整单报价、订单、担保、退款、唯一接取、到期恢复与平台介入共用后端状态；110 项后端测试含真实数据库并发、回滚与恢复，另 9 类 HTTP 响应通过契约校验。发布时未种入演示商品或委托；后续用户实测已生成真实业务，见下方记录。官方商品领取仍需真实游戏客户端验收。 |
| 管理网页 | 商店、模板、公告、案件与只读审计接入真实服务。当前 xiwanzi 有平台权限但尚无已创建的店铺，管理页如实提供创建入口。账号封禁及完整第三方 OIDC Provider 没有在本轮冒充已实现。 |
| 图片 | 雨云 S3 短时签名直传、完整图片校验、业务绑定和受限读取已接；Android / 浏览器实际上传、验证、读取、删除探测通过。存储凭据只在服务端受限配置。 |
| AI | DeepSeek V4 Flash、原有客服提示词的 IX 品牌修订、按需原生联网搜索；免费 20 次 / 24 小时，服务端管理员免次数额度，其余套餐 9999999.00 且禁购。真实 Android SSE、搜索来源及同请求恢复通过。 |
| 更新 | 实际 APK 下载、哈希、签名、版本及发布清单已核对；没有发布可执行代码资源包。 |

20003 基线 App 91 项测试通过、Lint 0 错误；20004 使用下述12项定向回归，未重跑全部测试。网页 54 项 Node 测试及多组浏览器契约/公网只读检查通过。00fbdac 基线 Go 全量 race / integration 110 项无失败，仅显式人工浏览器夹具 `TestSocialBrowserFixtureV2` 跳过。模拟器和协议测试客户端不能替代真实手机手感及完整游戏客户端切服确认。

## 游戏组件与最终更新

14:18 已验证 Login、Amiya、Odyssey 完成游戏启动，Core、Mail 与 XConomy 正常启用并连接后端。Amiya/Odyssey 上报 `playerDataReady=true`、provider=`youermodsync-transaction-v1`，Mail API 2 的 `commerceReady`、`uncertainClaimProtection`、`missingDeliveryCancellation` 均为 true。Login 禁领，MEK 安装同批文件后保持原停止状态及禁领设置。

| 组件 | 最终配套版本 | SHA-256 |
| --- | --- | --- |
| Core | 1.0.0 | `6be8b022e6a360b11455100150d59f1032087edd34c6d75fa8e4b81adf9c5137` |
| YouerModSync | 0.7-deuterium.3 | `376411646a903bcdb53808b68d94673f9a7f5ea9e47fde4e15957395517a9151` |
| 独立 Mail | 0.6.1 / API 2 | `9c8b4585799812e7874094d023e057acd059f9a0716210ac1b476ca009249cfa` |
| XConomy | 2.26.3-deuterium.1，已实服运行 | `880e9d5fa6b36c074e72df9ac250bc118bb80a62c949bbdadb203240228b6344` |
| Mail Bridge | 0.6.0，保持玩家协议 | `3f855586ae229d35e4a71d587be22a21b22969d1387e952aeb0532b743729d1c` |
| Mail UI | 0.6.0，保持界面 | `5634ab812ed3e8b2eae19b2c5f72952429208a656e4c3e3ddfd333c49104d966` |

Sync 库实际为三服共用的 `minecraft`：Amiya/Odyssey/MEK 的库存域应同为 survival，MEK 继续禁领并保留未验证兼容档案。Login 不启用玩家同步集成。玩家状态、精妙背包与保存证明同事务；未知领取只按原保存证明确认，不能重新发物。不存在邮件的退款需持久取消屏障，不能将 NOT_FOUND 当作可退款证明。

14:15 正常停服并确认原 Java/包装进程退出；14:16 完成三库同进程一致性快照及四服配置/玩家数据备份，14:17 换包，14:18 恢复三服。备份目录 `E:/Deuterium_IX/deployment-backups/app-v2-20260908/final-sync`，SQL 压缩包 SHA-256 `30eec7feff4554df32b7db4a3ef33df364dec9efcc7470f4e82b28758d9ee93b`；四个 ZIP 均通过 CRC 与逐文件摘要校验。仅关闭已证实无对应 MOD 的 cobblemon/armourers_workshop 休眠旗标，保留 playerdata/sophisticatedbackpacks 与原表数据。云端和四服的 MEK 库存域均对齐 survival，禁领不变。

启动脚本曾在读取过渡期状态时得到 `JSONDecodeError`，已保留 `start.failed.json`；后续只读确认所有原启动 ACK、三个游戏端口/协议及 Done 日志，记录为 `start-verified.json`，没有再次发启动命令。[安装证据](../qa/minecraft-final-2026-09-08/installed.json)、[备份清单](../qa/minecraft-final-2026-09-08/backup.json)、[启动确认](../qa/minecraft-final-2026-09-08/start-verified.json)。含私有数据的 SQL/ZIP 不公开下载。

## 用户实测与 14:25 热修

用户确认后续 ElmastorVox 转账、999998.00 市场订单购买后退款，以及 1.00 委托接取/提交是本人操作；这些是真实业务记录，不清理为演示数据。14:18 只读核对市场订单为 REFUNDED，委托为 COMPLETED、资金 HELD 等待发布者确认；原 13:47 的空态验收仅描述当时。

转账显示待确认的原因是 Go 校验 `fromUuid/toUuid`，而实际 XConomy 回执为 `payerUuid/payeeUuid`。热修 `8744f5b` 兼容两套字段，但所有出现字段必须与原交易一致，继续核对操作、金额、币种、状态及提交时间。相关 race 回归通过；部署后仅 GET 原 `transfer_732103cd41d23562ef51c6ebae75edfde4995431`，14:25 返回 HTTP 200 / success，原 Core 操作不变，没有再次执行付款。

20004 消费业务响应中已授权的图片 URL，避免对他人商品调用仅限所有者的通用资产接口；网络头像使用图片下载解码。Android 15 模拟器连接真实服务，成功解码他人商品图 512×288、ElmastorVox 头像 640×640，并通过原交易 GET 清除本机 pending、解锁下一次转账，全程无新业务写入；12 项定向测试通过。菜单改为点击客服头像，额度文字不再打开菜单；分类保留原位再次点击取消，移除重复叠层；注册及关于页统一版本标。后三项为局部 UI 改动与构建确认，没有扩大为新一轮全部 UI 测试。

## 源码与证据

- 部署集成：`C:/DeuteriumAPP/.worktrees/test-release-v2`；初始完整 Go 发布 `00fbdac`，当前钱包热修 `8744f5b`，Core / Sync 源码交接已并入。没有自动推送 main。
- Android：`C:/DeuteriumAPP/.worktrees/ui-motion-lab`，20003 基线源码 `ff0dcc3` / 只读 QA `215d9f2`；20004 热修源码 `a0aa7e4`，包含 UI 修正 `c89f4d4`、`93a800a`。交付目录的 `source-handoff` 保存精确文件清单；Mail 提供相对 0.6 + JDBC 基线的窄补丁，未混入其他未提交工作。
- 网页：`C:/DeuteriumAPP/.worktrees/web-player-v1`；静态源码 `4de1ee7`，公网 QA `2455abb`。
- [真实资金与节点验证](../qa/live-funds-2026-09-08.md)、[资金状态机验收](../qa/commerce-v2-2026-09-08.md)、[Core / Sync / Mail 隔离验收](../qa/core-sync-mail-2026-09-08.md)、[Core 严格取消契约](../contracts/mailbox-cancellation-core-v2.md)。

交付后首次使用建议完成一次原账号密码登录、真实手机操作，以及实际游戏客户端的领取/切服确认。临时 QA App/Web 会话已按确切 token hash 撤销，公网验证均返回401；用户自行登录的会话未受影响。
