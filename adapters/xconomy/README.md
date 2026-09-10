# XConomy 持久化资金扩展

当前目标：原版 XConomy-Bukkit 2.26.3 的指定输入，输出 `2.26.3-deuterium.3`。Java 21；适用于本项目 Youer 1.21.1 Bukkit 插件环境，不是代理端或 Sponge 构建。

2.0.9 候选新增可信插件 `rewardMail` 入口：固定 `mailcredit_<claim UUID>`，经既有原生变更事务提交并幂等重放，不加入网络执行白名单。配套 Core 1.0.2、Mail 0.6.2/API3；尚未表示生产部署。

扩展保留 XConomy 自有数据库与经济账号，由插件内部 API 完成受控资金事务。Core 只调用 `ControlledEconomyAPI`，不读取 XConomy 凭据、不直接更新经济表。

## 2.0.4 已发布增量

新增已提交账本的只读分页查询和北京时间当日汇总；`ledgerApiVersion()=1`，资金执行 API 仍为 1。Core 配套为 1.0.1。安装时幂等添加时间/玩家时间索引，既有原生与 App 流水均可读取，不重写经济余额。已部署四服，见[部署记录](../../docs/deployment/deuterium-2.0.4-live.md)与[统一流水契约](../../docs/contracts/admin-ledger-v204.md)。

## 并发与持久化

- 余额、担保快照、收款人绑定、资金流水和原操作回执同事务提交；提交响应丢失时，用原 operationId 查询或重放。
- `DataCon.changeplayerdata`、原 `/pay`、Vault 存取、XConomyAPI，以及旧 SQL.save/saveall 都经过同步事务。原 `/pay` 的两边余额一次提交，保留现有税率规则。
- 所有节点的余额读取直接查已提交数据库。Redis 缓存通知仍保留，延迟通知不能参与余额计算，也不能重新覆盖 SQL。
- 余额行按 UUID 排序加锁，扣款检查在锁内；不足时不会先扣一边。一次事务最多两次死锁重试，提交不确定不换操作标识。
- 当前使用 MySQL/MariaDB、InnoDB、`innodb_flush_log_at_trx_commit=1`、XConomy 连接池、两位小数模式。每笔金额和每个余额最大 `1000000000000.00`，同时遵守更低的 XConomy 上限。现有 DOUBLE(20,2) 列每次写入后核对精确小数值，不能无损存储则回滚。
- Redis 订阅遇到断线时按上限退避重连；停用时先关闭订阅连接并等待线程退出，再关闭池，避免卸载后访问插件类加载器。
- 新表使用 XConomy 表名加 `_dc_` 前缀；不改变旧余额列类型。新记账在 `_dc_ledger`，旧 xconomyrecord 不再作为此扩展的资金确认依据。

## 系统身份与托管

在经济权威节点的可信控制台执行 `/dc economy init-system-accounts`。先检查同名记录；发现冲突则拒绝接管。新建 DIMA、DaoYu 均为零余额，每个 UUID 持久化到 XConomy 自有系统账号表。

- DIMA：官方商城收入。DaoYu：实际托管余额。
- Core 拒绝这两个名称及其系统 UUID 登录游戏，后端拒绝普通注册登录，XConomy 拒绝普通存取、支付、删除、批量改款操作影响系统账号。
- reserve 从原付款人转到 DaoYu；官方收款人强制 DIMA；市场冻结卖家；委托接取人只允许绑定一次。
- settle 从 DaoYu 转到冻结收款人，refund 转回原付款人；单笔或累计金额不能超过余款。
- DaoYu 实际余额必须始终等于所有未结清担保余款。发现差异停止资金操作，不能用临时加钱掩盖。

## 构建与完整部署

先把经核验的原 JAR 安装为 Maven provided 依赖 `me.yic:xconomy-bukkit-input:2.26.3`，执行 `mvn package`，然后运行：

```text
python build_patch.py --input /path/XConomy-Bukkit-2.26.3.jar --output target/XConomy-Bukkit-2.26.3-deuterium.3.jar
```

脚本仅接受 SHA-256 `0e3695f75f9d8769bb365d6c60fe48d169baf162b0bed423b456acef0a53584f`，保留未修改内容并加入修改类清单；不覆盖输入。原源码来自 [XConomy](https://github.com/YiC200333/XConomy)，沿用 GPL-3.0-or-later；交付时一并提供本目录、LICENSE 和对应原版源码。

**四服必须作为同一轮完整停服更新**：等待旧异步任务退出、备份插件和数据库，再更换所有节点 JAR，验证版本后开放资金。不能热重载，也不能在仍运行旧写入路径的节点旁单独开放新资金 API。生产操作由部署任务统一执行。

## 验证

设置 `DEUTERIUM_FUNDS_TEST_URL=jdbc:mariadb://127.0.0.1:PORT/` 与本机测试密码 `DEUTERIUM_FUNDS_TEST_PASSWORD`，执行 `mvn test`。测试仅创建随机 `dc_funds_test_*` schema，结束后清理本次 schema；不读取生产配置。没有显式本机数据库时数据库测试会跳过，不能将跳过当通过。

已覆盖：转账幂等和不同参数拒绝、失败终态不翻转、真实系统账号预付/分结算/退款守恒、互斥绑定、结算退款竞争、双实例原生存取/pay/Core 并发、提交后丢响应、第二边 SQL 失败回滚、系统账号保护与托管余额差异停用。游戏内 Vault、原 `/pay` 和多服实测证据见后续 QA 报告。
