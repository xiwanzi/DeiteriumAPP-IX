# 聊天实时恢复、五分钟附近与路标显示上线

2026-10-13（UTC+8），**16:23 App 2.0.27（22700）与配套 Go 上线；16:29 游戏资源 0.0.58-dev 发布，16:30 完成公网核验**。聊天模组为 **0.1.10**。

主体源码 `9d010f8`，经 [PR #123](https://github.com/xiwanzi/DeiteriumAPP-IX/pull/123) 合并为 `669c8f7a782c919124493f38498109ef0d8b8606`。维护工作区 `.worktrees/game-chat-ldlib`，分支 `xiwanzi/chat-expiry-order`。

## 已生效内容

- 游戏附近文字、图片和贴纸从发送时间起满五分钟消失，页面停留期间自动清除，重载历史不续期；Go 附近媒体历史窗口同步缩短。
- 游戏私聊按完整发送时间排序，修复最新摘要先到、旧历史后到造成的乱序；保持 ID 去重。
- App 私聊实时推送，增加 WebSocket 客户端保活、退避重连、回前台/网络恢复补拉、公共聊天页五秒兜底。公共与私聊使用独立的成功同步位置补齐跨页缺口，摘要/实时包不提前推进该位置。
- 客户端 Xaero 默认配置只将 `waypoints_in_world` 从 `true` 改为 `false`；`waypoints_on_minimap`、`discovered_waystone_waypoints`、`other_waystone_waypoints` 保持 `true`。地图上的石碑标记保留，世界内所有路标的悬浮名称和图标关闭，包括玩家自建路标。

路标配置路径为 `config/xaero/minimap/profiles/default.cfg`；发布前后该文件仅一行变化。发布规格存于 `ops/client-config/xaero-waypoints.json`。

## 产物与验证

[App 2.0.27 下载](https://47.103.99.34/downloads/Deuterium-2.0.27-22700-test.apk)。包名 `com.deuterium.app.uilab`，非 debuggable 的 performance 包，证书 SHA-256 `04a213066a78a968c2a1328a153a0967f99d117e0a826285e3f4af705db6f2f3`，与旧版一致。

| 产物 | 字节 | SHA-256 |
| --- | ---: | --- |
| App 22700 | 9,072,194 | `12146a39b9f31f54cf7161c1d4b2b409388e22bc2a031996f76464c8fa9ce71a` |
| Go Linux amd64 | 21,999,776 | `a424b66d949b7c459f746c01439657478fcc12612387731568db46cd978120ee` |
| 聊天 0.1.10 | 18,932,880 | `5ce7237706421ef25fbe9b93403da70c7df396c4526c7f60fdb560b96b3e0875` |
| 游戏资源 0.0.58-dev TAR | 18,942,464 | `d21900042ad7a28735e8f2c39823571c8e8a4ad0f179ec3ae4aae958bb372d96` |

- 游戏 23 项、Android 173 项、Go 全包及 6 项定向数据库集成通过。Android 35 隔离回环验证实时接收、漏推补拉、公共/私聊各 125 条跨页恢复、去重及断线自动重连，详见[验收](../qa/chat-delivery-recovery-20261013.md)。
- 正式 APK 的 R8、资源压缩、lint、敏感值扫描和 22600→22700 覆盖安装通过，通过导出的桌面入口启动正常。
- 公网 APK 字节与哈希一致。旧版至 22699 获取 22700，22700/22701 不重复更新或降级；原 27 条发布记录保留。
- 公网认证 WSS 成功接入，私聊 opt-in 头被接受，ping/pong 正常。没有发送真实聊天消息，临时验收会话均撤销。
- 真实 DLauncher 更新器验证 0.0.57→0.0.58：替换聊天 JAR、更新资源显示版本和 Xaero 配置，旧聊天 JAR 被移除；既有字号、其他测试个人文件/模组/存档保留；再次更新无操作。
- 公网 HTTPS TAR 全量哈希、206 范围元数据及 TAR 内四项路标值均验证通过，原 72 条资源记录保留。
- 首次 OSS 同步在列举远程文件时发生 TCP 连接超时，未改变公开索引。保留失败记录后以新请求重试，`ls_d1c2c67bc330f927c394a8e84ee414d9` 成功，最终临时发布会话已撤销。

## 服务与数据

主 Go 切换到 `/opt/deuterium/releases/chat-recovery-669c8f7a782c` 并重启，健康检查 ready。没有数据库迁移，原迁移校验与配置哈希保持。切换前后 **61 账号、47 订单、5 委托、3,085 件在售商品、12 工单、2 注销记录**和既有业务 ID 集均保留。

社区 Web 2.0.38 无本轮页面改动，两个社区入口及 JS/CSS/Worker/WASM 公网资源校验通过。官网、客服站、独立图片审核保持，审核服务 PID 未变。

游戏服 Core/模组/配置及 PID 保持：Login、Amiya、Odyssey 在线，MEK 停止。无需游戏服重启。玩家更新 App，并退出游戏后通过启动器更新至 0.0.58-dev，即可使用本轮客户端修复。

## 备份与回退

Go/App 备份：`/var/backups/deuterium/chat-recovery-v227-669c8f7a782c`，包含一致性数据库备份、旧二进制、旧更新清单及配置。客户端备份：`/var/backups/deuterium/chat-recovery-client-0.0.58`，包含旧聊天 JAR、路标配置、资源索引、打包及同步记录。

回退只恢复对应程序链接/更新清单或本轮客户端文件与索引，不恢复整库覆盖发布后新增业务。App 本机交付在 `C:/DeuteriumAPP/delivery/Deuterium-2.0.27/`；客户端和节点验收证据在工作区忽略目录 `output/chat-recovery-release/`。

本轮没有额外发送账号更新提醒。模拟器与回环测试不等同真机运营商网络的固定延迟保证；实际 Wi-Fi/蜂窝切换体验仍以玩家真机为准。
