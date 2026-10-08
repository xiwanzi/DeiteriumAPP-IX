# Core 1.0.3 家具领取兼容性修复

## 问题与边界

2026-10-08，Amiya 的邮件 21 在 `lease.verifyItems` 阶段报 `Client does not advertise required mod: blahaj`，发生在占用领取事务及发奖之前。Youer `PlayerAPI.hasMod` 从客户端自定义网络通道命名空间构造集合，不是完整模组列表。Blåhaj 没有独立 payload 通道，因此不能以该集合判定物品兼容性。

## 实现

`ItemCompatibility` 继续检查每个物品的 NBT，仅在出现模组依赖时，对整批调用一次 `NeoForgeRegistrySync`。检测当前在线且已连接的 PLAY listener 必须仍然是连接实际使用的 listener；连接类型必须为 NEOFORGE；读取保留的 CONFIGURATION 协商结果，要求三个 `neoforge:frozen_registry*` 通道齐全。

这对应 NeoForge 21.1.248 的原生登录流程：`ConfigurationInitialization.configureEarlyTasks` 安排 `SyncRegistries`；客户端 `ClientPayloadHandler.handle(FrozenRegistrySyncCompletedPayload)` 在缺少注册表或未知条目时断开，成功后才回复；服务端 `ServerPayloadHandler` 才完成配置任务。不能把 PLAY 阶段的 `hasChannel` 当作配置通道检查，也不能只凭连接类型放行。

反射方法只解析一次；玩家、连接及判定结果均不缓存。不引入家具模组白名单、不要求额外客户端检测包，也不修改 Youer 的全局 `hasMod` 语义。接口不可用时失败关闭。

此检测适用于项目的 Youer 1.21.1 / NeoForge 21.1.248 原生注册表同步连接，不是对任意未来协议或恶意客户端的安装证明。其他加载器不会自动获得模组物品发放许可。

## 本次验证

- JDK 21、`mvn verify`：34 项测试通过。
- 新增 5 项测试：无独立通道和未来命名空间；批量只检查一次；原版物品；缺失同步通道/未完成配置/断线/非 NeoForge；重连重新判定；损坏 NBT 在兼容检查前被拒绝（相关断言分布在这 5 项测试中）。
- Amiya 与 Odyssey：加载候选 JAR 的隔离类加载器，只读解析实际运行的原生 API，均 `NATIVE_API_BINDINGS=PASS`。检测时无在线玩家，因此此项不是领取成功测试。
- 真实订单不代领、不重复下单、不改写领取状态。最终领取结果以玩家对原邮件重试的回执为准。

协议参考：[NeoForge 1.21.1 网络文档](https://docs.neoforged.net/docs/1.21.1/networking/)。同步任务及完成处理依据项目对应 Youer / NeoForge 源码中的上述类核对。

## 上线

2026-10-08，确认无在线玩家后，顺序更新并重启 Amiya、Odyssey。两服均加载 `DeuteriumCore v1.0.3`、完成数据库连接和 Core 启用，并到达服务器 `Done`。旧版 JAR 已分别备份；没有变更邮件领取标志、原订单、商品或交付模板。Login / MEK 的禁领配置未改变。

发布 JAR 的 SHA-256：`b899f27680b520dc5f3b057b8dbd21437167a0400ebaa500a9bf92a25075df43`。
