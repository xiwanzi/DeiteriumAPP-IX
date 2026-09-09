# Deuterium Core

2.0.4 候选版本为 **Core 1.0.1**：新增经济权威节点的只读 `wallet.records` 命令，配套 XConomy `2.26.3-deuterium.2`。原资金执行协议不变，当前尚未部署；见[统一流水契约](../docs/contracts/admin-ledger-v204.md)。

Java 21 / Youer 1.21.1 插件。主命令 `/dc`。物品库内置，邮箱仍为独立 DeuteriumMail；Core 只调用其公开 API。此目录不包含邮箱实现、经济数据库凭据或任意服务器命令执行接口。

## 物品库

| 命令 | 作用 |
| --- | --- |
| `/dc save <ID> [显示名]` | 保存手持物品的单件不可变快照，保留原物品 |
| `/dc get <ID> [数量] [版本]` | 领取当前或指定版本；整批容量检查，不掉落溢出物品 |
| `/dc give <玩家> <ID> [数量] [版本]` | 向本服已加载玩家发放并确认保存 |
| `/dc list [页码] [关键词]` | 分页查询物品目录 |
| `/dc info <ID> [版本]`、`versions <ID> [页码]` | 查看摘要、范围及历史版本 |
| `/dc policy <ID> <服务器,服务器> [单次上限]` | 生成新的交付范围版本，保留旧订单引用 |
| `/dc archive <ID>`、`restore <ID>` | 下架/恢复目录，保留旧版本 |
| `/dc sync`、`retry` | 重发目录元数据或重试被拒绝的出站事件 |
| `/dc status`、`mail`、`reconnect`、`reload` | 查看状态和邮箱能力、重连、加载允许热更的配置 |

默认命名空间为 `deuterium`。权限均在服务端核验，具体节点见 `src/main/resources/plugin.yml`。控制台 `/dc economy init-system-accounts` 仅在经济权威节点初始化/核验 DIMA 与 DaoYu 两个专用系统账号。

保存完整可独立还原的 ItemStack 字节，保留附魔、耐久、组件、PDC 和嵌套物品。已初始化的精妙背包包含外部存储 UUID，直接复制会共享同一份内容，因此明确拒绝入库与发放；尚未提供内容展开和新 UUID 分配的物品库适配。未初始化空背包及独立 NBT 物品仍按正常范围与兼容检查处理。

## 部署组合

- Core 物品库：显式 SQLite 单节点，或 MySQL 集群；数据库异常不降级到其他数据源。
- 跨服保存：[YouerModSync .3 适配](../adapters/youermodsync/README.md)。只有实际加载和事务保存确认后才允许领取。同步插件存在但启动失败时，禁止自动退回本机保存。
- 资金：[XConomy 持久化扩展](../adapters/xconomy/README.md)。所有四服统一更新；Core 不直接写 XConomy 表。
- 邮件：独立 DeuteriumMail `0.6.1` 的 API 2，包括创建、查询、撤回、未创建交付的取消凭据及保存证明恢复。版本号相同的候选包仍必须按交付 SHA-256 配套。
- WSS：Core 主动连接可信 HTTPS 后端，节点独立 token；公共聊天、目录、在线状态和邮件事件持久化去重。原始物品载荷留在 Core，App/网页只引用版本与摘要。

Amiya、Odyssey、MEK 的现有 Sync 共用同一 `player_data`，库存域均为 `survival`；MEK 继续禁领且保留独立未验证的兼容档案。Login 不参与玩家库存同步并禁领。禁领不等于另有独立背包数据库。

## 构建与证据

先安装同批独立 Mail 的 provided API，再使用 JDK 21 执行 `mvn verify`。测试探针 `integration-probe` 只允许回环地址与 `.tools/core-*` 隔离目录，不能部署到生产。

[物品及资金验证](../docs/qa/core-economy-2026-09-08.md)、[Sync 与独立邮箱验证](../docs/qa/core-sync-mail-2026-09-08.md)、[TrChat 公共消息验证](../docs/qa/core-trchat-2026-09-08.md)、[资金契约](../docs/contracts/core-funds-runtime-v1.md)、[保存证明契约](../docs/contracts/core-player-save-proof-v1.md)。生产部署和真实玩家验收由全栈部署任务单独记录，本机构建不等于已部署。
