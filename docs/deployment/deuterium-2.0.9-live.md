# Deuterium 2.0.9 部署记录

2026-09-10 **16:36:22（UTC+8）**，App **2.0.9（20901）**、Web 2.0.9、配套 Go、Core 1.0.2、Mail 0.6.2 / API 3、Mail Bridge 0.6.1、XConomy 2.26.3-deuterium.3 已合并并上线。

- [主仓库 PR #16](https://github.com/xiwanzi/DeiteriumAPP-IX/pull/16)，发布源码 `e4e4c4b7791c6e1c9888bccbf52dcaeadc761106`。
- [Mail PR #2](https://github.com/Deuterium-IX/Deuterium-Mail/pull/2)，合并源码 `67db93749bf97527ec3072214777c87e68e3132f`。保留其主分支已有的 UI 更新，实服 Mail UI 文件本轮不更换。
- [下载 App 2.0.9](https://47.103.99.34/downloads/Deuterium-2.0.9-20901-test.apk)，包名 `com.deuterium.app.uilab`。20900 是先前候选；正式发布使用 20901，确保候选设备也能收到更新。
- [优惠券管理](https://47.103.99.34/admin?section=coupons)：新建并保存草稿，在列表勾选后批量发放。
- 维护工作区 `.worktrees/commerce-v209`，分支 `xiwanzi/release-v209-live`。

## 已发布行为

支付成功页使用订单实际店名；成功结算后按已购买数量清理购物袋，失败/未知结果与后来加购保留。交付模板支持多个精确物品版本、数量和搜索分页。商品支持附带信用点、折扣及每玩家累计/每日/每周/每月限购。

优惠券支持全体或指定玩家、有效期、店铺/商品范围、商品折扣叠加、满减/折扣和封顶；每单自动选一张最优惠券。后台先保存草稿，勾选 1–100 张统一发布，一张校验失败时不部分发放。同批对每个账号只提醒一次，App 有合并浮窗、前台轻提示和“我的优惠”红点；断网确认保留并重试，过期券直接消失。

邮箱预览从原始物品快照读取组件；信用点与实物共享领取、保存证明和幂等恢复。Mail API 3 / creditRewards 已在可领取节点确认可用。没有向生产账号发放测试券、创建测试订单或测试奖励。

## 发布验证

- 主仓库更新部署记录引起的三份文档冲突已合并保留；业务子目录与已验收源码一致。Mail 合并主分支已有 UI/组件库更新，插件、桥接器、契约子目录与已验收产物源码一致。
- 发布前的 Go 全量竞态/集成、Android 135、Web 65、Core 29、XConomy 11、Mail 插件 164 / 桥接 11 / 契约 16 项验证见 [QA](../qa/commerce-v209.md)。版本码提升后重跑 Android 构建与 135 项单测，覆盖安装、桌面启动和签名验证通过。
- 公网完整 APK 与本机产物摘要一致。20004 至 20900 的已检版本均能获取 20901；20901/20902 不重复更新或降级。公开 HTML、JS/CSS 与网页包逐字节匹配，运行配置为 2.0.9，HTML `no-cache`。
- 用随后撤销的临时会话只读检查真实账号、钱包、21 笔当前账号可见订单及详情、商品、草稿/优惠券/提醒接口；21 笔可见记录不代表删除了其他记录，数据库仍保留全部 **22 笔订单、4 个委托**。70 个账号、既有 Saki 售卖开关和配置保留。
- Login、Amiya、Odyssey 已恢复，游戏协议探测与后端连接均正常；Amiya/Odyssey 报告 Mail API 3、commerceReady、creditRewards 可用。三服的玩家邮箱协议 3、管理员协议 6 网关已注册。MEK 保持停止、禁领。
- 四服 Core/XConomy/Mail/Bridge 摘要均匹配新包；所有原配置、Sync 以及原 Mail UI 摘要不变。临时验收会话已撤销，并确认返回 401。
- `xiwanzi` 账号已收到唯一 `app.release:20901` 更新提醒，没有广播公共聊天。

实际 TACZ 游戏客户端渲染、真实信用点领取/切服及真机手感没有在本轮发布中实测，不以接口或模拟器结果代替。

## 文件与备份

| 文件 | 字节 | SHA-256 |
| --- | ---: | --- |
| App 20901 | 23574133 | `f802da8962508a80d52211a177d433718902df865e47e2069f86be7ea9f8cffd` |
| Go Linux amd64 | 12996768 | `28067d019656b3b7e7c5cf2528b2eb96b1fa7eb07a812231129261a7b5f63fca` |
| Web 2.0.9 ZIP | 33695546 | `8815c9d7077e16d84b30cf54363b9c6f20480c53955f6dfe1505a5a58e7c07c6` |

完整组件摘要与实际构建提交见 [构建清单](artifacts/v209/build-manifest.json)。APK 与 20800 同签名，证书 SHA-256 `04a213066a78a968c2a1328a153a0967f99d117e0a826285e3f4af705db6f2f3`。Go 直接从合并源码构建，`vcs.modified=false`。

Go 位于 `/opt/deuterium/releases/release-2.0.9-e4e4c4b7791c/deuterium`，Web 位于 `/var/www/deuterium/releases/web-2.0.9-e4e4c4b7791c/dist`。

云端私有备份 `/var/backups/deuterium/release-2.0.9-20260910-e4e4c4b7791c`，包含配置/环境/原更新清单、systemd/Nginx 配置、旧程序/网站指向，以及在线与停服后的 SQL gzip。最终停服快照 1,084,276 字节，SHA-256 `287e60cf29de061408b80c97bc944a3cfb45d0e137275a2c7e84cb5b44b9dee2`。

游戏私有备份 `E:/Deuterium_IX/deployment-backups/app-v209-20260910-20901`。16:23 正常停服，确认 Java/包装进程全部退出后备份三库与四服插件/配置/玩家状态；三库 SQL gzip 198,339 字节，SHA-256 `6b81f3ed3f3ab2a49c2b8bb60e72d1a5f57e6c7d487f6d7b29ed17ddeb12e610`。四个归档的 CRC、每个文件摘要均通过，旧 JAR 保存在该目录；16:28 三常驻服已恢复。备份和凭据不在公开下载或源码中。

新增 023–025 迁移通过 root Unix socket 维护连接执行，原迁移摘要逐一一致，记录数为 21。运行账号没有扩大 DDL 权限。保留所有新增表、券使用/提醒/批次记录；回退不能覆盖整库或抹除后续业务。有新型信用点邮件或优惠订单时保留兼容恢复能力，已安装 20901 的设备只能用更高版本码修正。

[云端备份](artifacts/v209/cloud-prepared.json) · [停服快照](artifacts/v209/cloud-paused.json) · [游戏备份](artifacts/v209/game-backup.json) · [游戏恢复](artifacts/v209/game-final.json) · [发布结果](artifacts/v209/published.json) · [公网检查](artifacts/v209/public-verification.json) · [真实接口](artifacts/v209/backend-readonly.json) · [最终状态](artifacts/v209/final-state.json)
