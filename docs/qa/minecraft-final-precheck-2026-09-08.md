# 最终 MC 换包前只读预检

2026-09-08，数据库观测时刻 **05:33:28 UTC / 13:33:28 Asia/Singapore**。对象为 SSH 别名 `ovo-server-codex` 对应的 MC Windows 主机。本次仅读取 JAR、配置中的非敏感范围、数据库聚合统计和本地世界目录大小；未启停节点、未替换 JAR、未写业务数据库，也未改客户端。

## Sync .2 基线

四份当前 JAR 均为 **97,513 bytes**，`plugin.yml` 版本 **0.7-deuterium.2**，SHA-256 一致：

`14fc3674b38153ec3db2c56fa8ee70a9662e854e7f8e5fdec9c962fb5dc059f4`

| 节点 | 精确 JAR 路径 | playerdata / sophisticatedbackpacks |
| --- | --- | --- |
| Login | `E:/Deuterium_IX/servers/Deuterium_Login/plugins/YouerModSync-0.7-deuterium.2.jar` | false / false |
| Amiya | `E:/Deuterium_IX/servers/Deuterium_Amiya/plugins/YouerModSync-0.7-deuterium.2.jar` | true / true |
| Odyssey | `E:/Deuterium_IX/servers/Deuterium_Odyssey/plugins/YouerModSync-0.7-deuterium.2.jar` | true / true |
| MEK | `E:/Deuterium_IX/servers/Deuterium_MEK/plugins/YouerModSync-0.7-deuterium.2.jar` | true / true |

配置均位于各节点 `plugins/YouerModSync/config.yml`，指向本机 `localhost:3306/minecraft`，四节点 `modules.vault=false`。Amiya/Odyssey/MEK 在新 `.3` 中同为 survival 的安排是本轮已确认部署输入；`.2` 配置没有该新域字段，不能把计划写成已经生效。

## 数据库实测

实际数据库服务是 **MySQL 8.0.45**，`innodb_flush_log_at_trx_commit=1`。本机使用的 MariaDB 12.1 客户端工具名称不代表服务端版本。

| minecraft 表 | Engine | 精确 COUNT(*) | 分配的数据 bytes | 分配的索引 bytes | 合计 |
| --- | --- | ---: | ---: | ---: | ---: |
| player_data | InnoDB | 2 | 16,384 | 0 | 16 KiB |
| backpack_data | InnoDB | 0 | 16,384 | 0 | 16 KiB |
| history_data | InnoDB | 93 | 1,064,960 | 16,384 | 1.03125 MiB |

`history_data` 的统计估算行数为 82，实际 COUNT 为 93；表中已明确采用实际 COUNT。它另报告 `DATA_FREE=4 MiB`。DATA_LENGTH / INDEX_LENGTH 是分配空间估算，不等于逻辑备份文件大小；InnoDB 的 TABLE_ROWS 也不是精确行数。[MySQL TABLES 元数据说明](https://dev.mysql.com/doc/refman/8.0/en/information-schema-tables-table.html)

| 整库备份范围 | 表数 | Engine | 数据+索引 bytes | MiB |
| --- | ---: | --- | ---: | ---: |
| minecraft | 34 | 全部 InnoDB | 2,244,608 | 2.140625 |
| deuterium_core_v2_test | 8 | 全部 InnoDB | 147,456 | 0.140625 |
| deuterium_mail_v2_test | 22 | 全部 InnoDB | 589,824 | 0.5625 |
| 三库合计 | 64 | 全部 InnoDB | **2,981,888** | **2.84375** |

实例全部非系统 schema 的数据+索引估算总和约 **95,911,936 bytes / 91.46875 MiB**，包含其他业务和示例库，未建议将其混入这次三库回滚。`.3` 新增 `dc_sync_sessions`、`dc_sync_save_receipts`、`dc_sync_recovery_audit` 在当前 minecraft 中的存在数为 **0**。

只读连接从当前 **Amiya 的 Sync 私有配置**读取凭据成功；旧 App 私有配置连接得到 1045，不能继续当作有效备份入口。复用脚本为 `C:/DeuteriumAPP/.tools/deployment/final-sync-readonly-db-current.ps1`，通过现成 `mc_read.py` 调用。脚本不输出凭据；临时客户端 cnf 仅位于受限 CodexStaging，使用后删除。MySQL 8 不支持 MariaDB 的 `max_statement_time`，脚本没有使用该变量。

## 一致性备份建议

最终让四节点和能执行交易的后台停止写入后，用**同一个 dump 进程**以 `--single-transaction --databases minecraft deuterium_core_v2_test deuterium_mail_v2_test` 创建一个共同快照，并保留二进制字段、触发器及所需 routine/event 定义。三个库总量很小，整库备份比挑同步表更容易保证 XConomy 担保流水、Core 原操作、邮箱交付与同步状态的关联。备份完成之前不启动 `.3` 创建表或执行其他 DDL；MySQL 明确指出单事务快照只保证事务引擎，并要求备份期间避免 ALTER/CREATE/DROP/RENAME/TRUNCATE。[MySQL mysqldump 一致性说明](https://dev.mysql.com/doc/refman/8.0/en/mysqldump.html)

同时保存四服原 Sync/Core/Mail JAR、相关配置及停止后的 `world/playerdata`、`world/data`、`world/advancements`、`world/stats`、`world/level.dat*`；如要回退涉及区块内容的游戏领取或模组全局状态，保留完整世界备份。四服配置 `level-name` 均为 `world`。已确认 Login 与 Amiya 的 `world/data/sophisticatedbackpacks.dat` 存在；Odyssey/MEK 的该层目录未见同名文件。MEK 当前 playerdata 目录为空。

目录中的直接文件数量不等于玩家数，例如 `.dat_old` 也计为文件。本次没有读取 NBT 内容。停服、备份、校验备份可恢复性、换包及启动验收由主发布任务执行，本报告不是备份完成凭证。

## 客户端最小 JAR 清单与候选

本轮 Mail API 2 变化发生在服务端插件/桥接；现成 Mail UI 候选仍为 **0.6.0**，与四服现在安装的 UI JAR 哈希一致。本次未发现另一个待分发的新 UI JAR，也未核对用户实际客户端目录。若客户端缺失或过旧，最小清单是 Mail UI，以及缺失/低于要求的两个界面依赖；已经满足要求的依赖可保留。

| 用途 | 现成候选 | SHA-256 |
| --- | --- | --- |
| Mail UI 0.6.0 | `C:/Mod/DeuteriumIX/work/mail/Deuterium-Mail/mail-ui/build/libs/deuterium-mail-ui-neoforge-1.21.1-0.6.0.jar` | `5634ab812ed3e8b2eae19b2c5f72952429208a656e4c3e3ddfd333c49104d966` |
| UIKit 0.1.11，modId=deuterium_ui | `C:/Mod/DeuteriumIX/work/mail/Deuterium-Mail/vendor/ldlib2-uikit/0.1.11/ldlib2-uikit-neoforge-1.21.1-0.1.11.jar` | `3db59c00d4830d41944ef72f4299c750b880c133c9db5217ca5fff87233db3cf` |
| Mail UI 编译采用的 LDLib2 2.2.37 | `C:/Users/34545/.gradle/caches/modules-2/files-2.1/com.lowdragmc.ldlib2/ldlib2-neoforge-1.21.1/2.2.37/9f144c0d89f88ba16a656bff181143a2b5e4dc6b/ldlib2-neoforge-1.21.1-2.2.37-all.jar` | `41df4b79f0a3ec622f221c75e5315988d27a5c8896f3ed80993d5f87f7bdbcde` |

实际 UI 描述文件要求 Minecraft 1.21.1、NeoForge `>=21.1.221`、客户端 LDLib2 `>=2.2.37`、客户端 deuterium_ui `>=0.1.11`；构建使用 NeoForge 21.1.238 / Java 21。以上是读取本地构建文件及依赖描述所得，未据此声称所有更高版本均已真机测试。

14:13 补充传递依赖核对：deuterium_ui 0.1.11 自身要求 NeoForge `>=21.1.238` 与 LDLib2 `[2.2.37,2.3)`，所以整套客户端最低要求按这个交集，不能只引用 Mail UI 单包的 21.1.221。优先保留现有整合包中已验证的 LDLib2 2.2.38.a 原件；同版本号的本机缓存哈希不同，不自动替换或混发。Core、Sync、独立 Mail 插件及 SDK 不安装到玩家客户端。

四服现在 `mods/ldlib2-neoforge-1.21.1-2.2.38.a-all.jar` 的 SHA 为 `8aa7d5e6de7960214eb6457c6c9744ad09a3395b62e6bdd87a4e1657086c3163`，大小 7,330,470 bytes；本机 Gradle 同名 2.2.38.a 缓存 SHA 为 `824ed41de63080d6337fe4907ba81a41446e25b5badd746798d36a0af685fb12`，同大小但字节不同。因此不能仅按同名/同版本把缓存包替换到已有客户端；若要统一四服当前 2.2.38.a，应核对所分发原件的精确哈希。

`Deuterium-Core`、`deuterium-mail` 业务插件、`deuterium-mail-bridge` 和 `YouerModSync` Bukkit 插件 JAR 属于服务端，不列入客户端 mods。现成 Mail UI 已包含自身协议内容，不额外分发独立 mail-contract JAR。
