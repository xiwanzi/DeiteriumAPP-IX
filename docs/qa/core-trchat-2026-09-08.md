# Core / TrChat 最终公共消息验证

2026-09-08 13:57，本机隔离 Youer 1.21.1、TrChat 2.4.9、PlaceholderAPI 2.12.3。Core 后端桥关闭，只使用回环测试玩家 CoreProbePeer 和专用 dc_core_game_* 数据库。

TrChat 的 BukkitProxyEvent 子类共享 HandlerList。Bukkit 原始 EventExecutor 不自动筛选具体类，首次实际运行发现 TrChatEvent 也进入了最终消息监听器；Core 现先检查实际事件类型，再读取 TrChatSendEvent 字段。该修复只涉及聊天事件入口。

## 实际验证

通过真正的 TrChat 类构造并派发最终事件，验证 HIGHEST 优先级替换、取消发送、SENDER、RECEIVER、Private 频道及真实 PlayerData 暗禁言状态；另外调用测试玩家 Player.chat，经过 Bukkit → TrChat 的限次、过滤、格式化和最终事件链路。

| 场景 | Core 持久队列结果 |
| --- | --- |
| 公开事件先传 RAW，HIGHEST 改为 APPROVED | 只保存 APPROVED，未读取替换前内容 |
| 已取消的公开事件 | 0 条 |
| SENDER / RECEIVER 私聊事件 | 各 0 条 |
| COMMON 类型的 Private 频道 | 0 条 |
| PlayerData 真实暗禁言状态 | 0 条 |
| 真实 Player.chat 公共发言 | 恰好 1 条，保留 TrChat 最终格式化文本 |

本次标记为 `DC_CHAT_48f5da18460d4b8d8d1ce841cd982af1`。对应 SQL 队列总计恰好 2 条：APPROVED，以及 `[core-test-world] CoreProbePeer: …_PIPELINE`。没有私聊或过滤前内容；修复后的本次日志没有事件类型反射错误。

Core `mvn verify` 29 项全部通过、0 失败、0 跳过。实际验证 JAR SHA-256：`6be8b022e6a360b11455100150d59f1032087edd34c6d75fa8e4b81adf9c5137`。与此前 `388534…` 的生产代码差异仅为聊天事件类型守卫；Sync/Mail/XConomy 配套 JAR 不变。

探针位于 `deuterium-core/integration-probe`，仅能在回环隔离服由控制台执行 `dcprobe trchat`。私聊及取消场景验证的是真实事件边界，未宣称跑过完整玩家私聊命令或代理网络。App/WSS 运输验证和生产端验收属于部署任务的证据，不能用本机队列测试替代。
