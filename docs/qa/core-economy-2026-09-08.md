# Core 物品库与持久化资金验证（2026-09-08）

本报告对象为 `backend-rewrite` 工作线本机构建。游戏验证使用独立回环 Youer `1.21.1-f15a736d`、Java 21、Vault `1.7.3-b131` 和专用随机测试数据库；参与者是 `CoreProbeUser`、`CoreProbePeer` 两个自动化测试客户端。未冒充生产玩家，未修改生产经济数据。生产安装由部署任务另行记录。

## 物品与保存

- 测试服分别以 SQLite、MySQL Core 存储成功启动；MySQL 使用 Core 私有 `cafe.deuterium.core.libs.mariadb.Connection`。
- 钻石剑的附魔、耐久、自定义名称、Lore、PDC 字节，以及潜影盒中嵌套物品的 NBT，在 capture → 存储 → decode 后保持一致。
- 真实 `/dc save`、`/dc get` 成功，保存保留原手持物品；物品库版本独立存储。
- 背包无法容纳请求数量时整体拒绝，既有物品未改变。
- 单节点保存证明实际执行玩家存档保存，重新读取 `.dat` 中 Inventory 与内存逐项比较，并刷新文件到磁盘；没有把 void saveData 返回当作证明。
- 死亡或失效玩家已被禁止进入发放保存屏障。此报告不把单节点证明当作跨服 Sync 证明。
- 离线目录测试主动移除了专用库中 CoreProbePeer 的 dc_players 行，玩家保持离线；`player.resolve` 仍从 Bukkit 已有缓存返回同一 UUID 与真实 lastSeen，没有计算新 UUID 或创建 App 身份。

本机最终附魔剑 SHA-256：`1968623c49ba57e4c79d2ed48079c8c63673143113fc3172e544829a7e544ebf`；嵌套容器：`a8e80bc2ca234bdf20b86dcb8915dc2f05d77c1900b6f88ae27847a6fdf8253f`。探针源代码位于 `deuterium-core/integration-probe`，具有回环与隔离目录检查，禁止部署到生产。

## 资金事务

XConomy 扩展 8 项真实数据库测试通过，验证：

1. 同操作重放不重复扣款，同键不同参数拒绝；已失败操作不会因后来充值变为成功。
2. DIMA/DaoYu 初始化为真实零余额经济账号，UUID 持久化；官方预付、分次结算与退款总额守恒。
3. 委托收款人只绑定一次；结算和退款竞争时总额受剩余担保款限制。
4. 两个执行实例同时处理旧存取、旧 pay 和 Core 转账，不使用旧缓存覆盖余额；1000 信用点竞争 60 次 30 点扣款，只成功 33 次，余款 10 点。
5. 数据库已提交但响应丢失时，查询与重放原 operationId 找回同一回执，不重复扣款。
6. 第二条余额写入故障时，第一条扣款、流水与操作记录全部回滚。
7. 系统账号不能普通改款或删除，未结清担保关联账号不能删除；DaoYu 余额不等于担保余款时停止资金操作。
8. 外部金额拒绝科学计数法、超界与超过两位小数的输入。

游戏内另外直接验证了 Vault deposit/withdraw、原 XConomyAPI、Core 同 operationId 转账重放、原 `/pay`。第二轮 Core 转账回执 ID 为 `live_dbfa2402-9339-41c9-be2a-41ed3aec06d4`，随后 `/pay 1.10` 后两账号余额为 `191.80 / 7.20`，符合测试前余额与本轮全部收支。DIMA 名称登录在登录阶段被拒绝。

## 驱动与关闭生命周期

- Core 私有驱动 JAR 检查确认认证 SPI 资源已随包名迁移，包含 caching_sha2_password 工厂。
- 先注册抢占同一 JDBC URL 的测试驱动，并设置 platform TCCL，Core 仍通过自己的 Driver.connect 成功连接本机数据库，且恢复原 TCCL。
- RSA 公钥获取仅对明确回环主机开放；没有降级数据库账号认证。MySQL 8.0.45 的生产验证由部署任务报告已通过。
- Redis 测试使用独立真实 Redis 8.10.1 回环实例，已观察到订阅线程运行。12:01:29 正常停服后 XConomy 成功停用，没有卸载异常或 Redis 线程的 NoClassDefFoundError。修复顺序为停止订阅连接、join 线程，再关闭连接池；避免重复 unsubscribe 已关闭连接。

## 本报告的界限

Sync 完整跨服领取、模组外部存储引用物品、生产玩家挑选物品和实际领取仍需相应验证。邮箱保持独立；Core 资金实现通过不等于 Mail 已具有同步屏障或生产订单已经完成。
