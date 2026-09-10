# Deuterium 2.0.8 部署记录

2026-09-10 11:13:20（UTC+8），**App 2.0.8（20800）、Web 2.0.8 与配套 Go 已合并发布上线**。

- [PR #14](https://github.com/xiwanzi/DeiteriumAPP-IX/pull/14) 已合并，发布源码 `6566bb617f8d079723bfd0c4876ece598d89ee36`。
- [下载 App](https://47.103.99.34/downloads/Deuterium-2.0.8-20800-test.apk)，包名 `com.deuterium.app.uilab`。
- [管理端图标切换](https://47.103.99.34/admin?section=launcher-icon)，仅平台管理员可操作。
- 当前维护工作区 `.worktrees/launcher-icons-v208`，分支 `release/v208-live`。

## 本次发布

默认使用无横幅的白底黑色弧线 D，内置“911双子节”周年活动图标。管理网页可预览并发布选择，在线 App 收到提示后补拉配置，启动和前台轮询补同步。图标只在已安装 APK 的内置款式之间切换，新增款式仍需随新版 App 发布。

线上初始配置保持 `default`、版本 1，没有为验收而切换生产用户的活动图标。022 迁移仅新增图标配置表；已应用迁移总数 18。原配置及环境文件摘要不变，22 笔订单和 4 个委托保留。游戏插件未修改或重启。

## 发布经过与确认

用户明确要求快速合并发布、无需再测试。本次沿用上一轮已完成的 [129 项 Android 单测、60 项 Web 单测、Go 竞态/集成和原生/网页验收](../qa/launcher-icons-v208.md)，**没有重跑测试**。App/Web 构建输入与合并源码一致；Go 由合并源码构建，`vcs.modified=false`。

首次启动因遗漏显式迁移而未就绪，发布脚本自动恢复了 2.0.7 程序、网页和更新清单。运行账号没有 DDL 权限，随后通过本机 root Unix socket 维护连接执行既有 `migrate` 命令，再完成发布；没有扩大运行账号权限，也没有恢复 SQL 覆盖业务数据。

最终仅做发布完成确认：服务 ready，公网 APK 可下载且大小正确，Web 配置版本 2.0.8，公共图标配置为 `default`，旧 App 更新接口已提供 20800。保留之前的测试证据，不把本次部署确认描述成新一轮测试。

## 文件与备份

| 文件 | 字节 | SHA-256 |
| --- | ---: | --- |
| Deuterium-2.0.8-20800-test.apk | 25062134 | `62fd90d6cf2c7842283bb77e410d8e299560d169ef711d69637a7d693fca9bfa` |
| web-2.0.8.zip | 33685205 | `2ac3933a8512e4fcd811b8dd70f421107d9a8659738835c16eb201dc4347609a` |
| deuterium-linux-amd64 | 12849312 | `d46df3c7650530046166592a4e4531d58fa4bea83e57d5d9d4fc5b8d8bb20280` |

APK 与 20700 同签名，证书 SHA-256：`04a213066a78a968c2a1328a153a0967f99d117e0a826285e3f4af705db6f2f3`。

Go：`/opt/deuterium/releases/release-2.0.8-6566bb617f8d/deuterium`。Web：`/var/www/deuterium/releases/web-2.0.8-6566bb617f8d/dist`。

备份目录：`/var/backups/deuterium/release-2.0.8-20260910-6566bb617f8d`。包含一致性 SQL gzip、原配置/环境/更新清单、systemd/Nginx 配置及旧程序/网页指向；数据库备份 870858 字节，SHA-256 `2cc6820a9044f43dbaaa13e4f5e516add625415031c3073304810d116cec9604`，仅保留在服务器私有目录。

外观回退通过后台发布更高版本的 `default`；不得降低图标配置版本或整库回滚覆盖新业务。旧 APK、程序和网页目录保留。已安装 20800 不会自动降级；如回退到没有图标接口的旧 Go，客户端会保留最后有效图标，应保留兼容接口或先完成默认图标同步。

证据：[构建](artifacts/v208/build-manifest.json)、[发布前状态](artifacts/v208/preflight.json)、[备份](artifacts/v208/prepared.json)、[发布](artifacts/v208/published.json)、[公网确认](artifacts/v208/publication-confirmed.json)、[更新接口](artifacts/v208/update-feed.json)。
