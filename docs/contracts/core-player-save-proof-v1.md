# Core / Sync / 独立邮箱保存证明

2026-09-08。Core 与 Mail 各自定义公开接口与 record，由 Core 的 MailBarrierAdapter 转换；不共享邮箱内部实现或数据库。Sync 同时保留不可变 SQL 保存回执。

```java
default SaveProof lookupSaveProof(UUID playerUuid, String inventoryDomain,
                                 UUID operationId, String sessionEpoch) throws Exception;
record SaveProof(UUID playerUuid, String inventoryDomain, UUID operationId,
                 String sessionEpoch, String serverId, String saveReceipt,
                 long committedAt) {}
```

这是线程安全的只读查询，不访问在线玩家、不重放物品、不改变会话或邮箱状态。四个请求身份字段必须精确一致；serverId 是原保存节点，committedAt 是数据库 epoch 秒。null 表示找不到或不支持，**不能解释为未发放**；查询失败抛异常。后端/邮箱不能接受客户端自报证明。

领取顺序：Sync 确认完整加载 → 持久化 LEASED 和 operationId → 独占玩家行锁 → 校验物品与容量 → Mail 固定领取意图 → 写入背包 → Sync 在同一事务保存全部模块、player_data 和回执 → 提交并读取校验 → Mail 提交 CLAIMED 与事件 → 释放玩家屏障。只读 `/dc save` 未改背包时释放屏障，不写出虚假发放回执。

Sync 保留 `youermodsync-v1:<operationUUID>:<sessionEpoch>:<snapshotSHA256>`。SHA-256 覆盖带长度分隔的玩家 NBT、capability NBT、成就数据；外部背包存储与该回执同事务提交。不能单凭“save 方法返回”或固定等待时间产生回执。

若 Sync 已提交而传输确认丢失，保存证明仍可按原操作查到。Mail 仅凭自身原领取意图与匹配的正向证明恢复 CLAIMED，且在自己的事务内出原领取事件；不会再次给予物品。若没有正向证明，继续 UNKNOWN，不能退款或重发。Mail 还需要防止一个 operationId 被不同邮件复用；该约束属于 Mail 自有意图与恢复实现。

若 Sync 在发放中遇到未知结果，其玩家会话会隔离并踢出，普通退出不覆盖保存。运维先执行 `dcsync inspect <playerUUID>` 与 `dcsync receipt <operationUUID>` 核对，再处理独立邮箱的原领取记录。玩家离线、租约过期后，可在控制台执行 `dcsync recover <playerUUID> <原epoch> accept-persisted` 接受数据库现有快照；该操作写恢复审计，不创造保存证明，不修改 Mail。错误 epoch、未过期会话或在线玩家拒绝恢复。

新过期 RPC 拒绝执行；同类型与指纹的旧操作过期重放返回原成功/失败回执，UNKNOWN 仍保持 UNKNOWN。过期不能把已经发生的扣款或发放改写成“本次未执行”的失败。
