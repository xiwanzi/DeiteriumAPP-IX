# 2.0.0 实服资金与节点验收

验收时间：2026-09-08 12:18，Asia/Singapore。本记录只证明下述已执行范围，订单/委托完整状态机与 Sync 跨服领取仍在联调。

## 部署对象

- 云端入口：`https://47.103.99.34`。后端目录 `/opt/deuterium/releases/funds-stage-becbc0e`，二进制 SHA-256 `a0a2d2b08aa59ff9a38a79fabcd1cf981236cebc3aaadd6f1bc217c7a007a0f3`。
- 编译源为部署分支已提交 `becbc0e` 的独立快照；72 项 Go 测试通过（含 race 和隔离 MariaDB），手工浏览器夹具 `TestSocialBrowserFixtureV2` 因未显式启用而跳过，其余业务测试未跳过。
- Core `1.0.0` SHA-256 `dd2c6a3b902b6afc6bc503d4b3ff6c17dafb3f53202628f6f76642fa0e2eae41`；XConomy `2.26.3-deuterium.1` SHA-256 `880e9d5fa6b36c074e72df9ac250bc118bb80a62c949bbdadb203240228b6344`。
- 四服按同一轮停服更新了 XConomy，并安装 Core、独立 Mail、Mail Bridge 与 Mail UI。原 XConomy 哈希四服均为 `0e3695f75f9d8769bb365d6c60fe48d169baf162b0bed423b456acef0a53584f`；修改前保存插件/配置及停服后的数据库快照。
- Login、Amiya、Odyssey 已正常恢复并上报在线，MySQL 连接池、Redis、Vault/XConomy 及 Core/Mail 启用日志已核对。MEK 已安装但保留本轮开始前的停止状态，不能记为在线验收。
- 四服原 `usepool=false` 已改为 true；保留既有两位小数。实际数据库为 MySQL 8.0.45，XConomy 表 InnoDB、`innodb_flush_log_at_trx_commit=1`。
- `deuterium-core/integration-probe` 仅用于隔离测试，未安装实服。

## 系统专户

在 Amiya 可信控制台执行 `dc economy init-system-accounts`，SQL 只读复核 DIMA 和 DaoYu 均已写入 XConomy 系统身份表，初始余额 `0.00` 且 `hidden=1`。没有为其创建 App/QQ/密码账号，没有用控制台补款冒充托管资金。普通登录与经济操作的保护由 Core 和 XConomy 扩展执行。

## 授权的小额转账

用户明确指定付款 xiwanzi、收款 luoyinwuchen1、管理权限 xiwanzi；允许付款余额不足时补足，本次无需补款。

| 项目 | 实际结果 |
| --- | --- |
| 金额 | 1.00 CREDIT |
| xiwanzi | 99,974,286.00 → 99,974,285.00 |
| luoyinwuchen1 | 799.00 → 800.00 |
| 业务请求 | `qa-live-transfer-20260908-01` |
| transferId | `transfer_acad3a87845dcba74310068ae5ab734d791a7de1` |
| Core operationId | `coreop_258c2ad398e509840323eaf64e2883f9c686d1cc` |
| 原请求重放 | 返回相同 transferId / operationId，无第二笔扣款 |
| 同键改额为 1.01 | HTTP 409，未执行 |
| App Bearer / Web Cookie | 读取同一成功回执与可用账单 |
| 经济账本 | 1 个操作，恰好两条账目：xiwanzi -1.00、收款人 +1.00 |

收款人当前离线且未注册 App，查询仍由实服既有玩家缓存解析到正式 UUID。没有盲算 UUID，没有为了收款伪造 App 身份。本文的“两端会话验证”为真实 API 调用；原生 App 和浏览器页面的显示验收另由各端记录。

## 备份与剩余验证

MC 本轮备份根为 `E:/Deuterium_IX/deployment-backups/app-v2-20260908/economy-rollout`，其中 `sql-stopped` 保存停服后经济、Core、Mail SQL，`before.json` 记录原运行状态，`installation.json` 记录安装哈希。云端受限备份位于 `/var/backups/deuterium/pre-commerce-20260908`。备份含私有数据与配置，不随公开 APK 或网页交付。

仍须完成 Mail 的不存在交付严格取消证明、真实 Sync 保存屏障、邮箱领取/撤回竞争与三端订单/委托验收。不能把本次转账成功写成官方商品已能完整交付，或把安装成功写成跨服领取已通过。
