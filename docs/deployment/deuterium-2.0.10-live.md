# Deuterium 2.0.10 部署记录

2026-09-11 **02:38:40（UTC+8）**，App **2.0.10（21000）**、Web 2.0.10 与配套 Go 已上线，`xiwanzi` 账号已收到唯一 `app.release:21000` 更新提醒。

- 发布源码：`65c08a290896601c6254369792e4223216ebee8e`；功能 [PR #18](https://github.com/xiwanzi/DeiteriumAPP-IX/pull/18)、版本/交付 [PR #19](https://github.com/xiwanzi/DeiteriumAPP-IX/pull/19) 已合并。
- [下载 App 2.0.10](https://47.103.99.34/downloads/Deuterium-2.0.10-21000-test.apk) · [优惠券管理](https://47.103.99.34/admin?section=coupons) · [商店管理](https://47.103.99.34/merchant)。
- 工作区 `.worktrees/commerce-refunds-performance`，分支 `xiwanzi/release-v210-live`。

## 生效行为

优惠券到账统一使用应用内顶部通知，单券/多券不再以大弹窗打断操作；关闭提示保留“我的优惠”红点，实际查看后确认。沿用批次去重、离线确认及账号隔离。

后台可以删除草稿/停用券，以及商店商品。删除商品同时停止销售、移出经营列表，原订单/交付/退款/审计记录保留，退款恢复库存不会重新上架。启用券需要先停用再删除。

商城全额退款证明确认后恢复原券资格，包括实付为零的订单；失败、未知结果或部分退款不释放。原有效期、启停和受众规则仍生效，重复旧退款不影响新订单重用。

开屏徽标改为异步加载、缓存复用；开屏/钱包刷新/存储加载动画将状态读取放到绘制阶段；顶部模糊缩小采样区域，减少每秒校时造成的优惠红点更新。骁龙 8E 真机手感仍待用户实测，未报告具体加速比例。

## 实际发布验证

- Go 从合并源码构建，`vcs.revision=65c08a290896…`、`vcs.modified=false`。App 提升版本后的 134 项单测与构建、Web 65 项测试与构建在本轮此前完成；Go 全量竞态/集成及原生 7 组交互结果见[验收](../qa/commerce-refunds-performance.md)。此次部署未重复执行这些业务测试。
- 已验证公网完整 APK 的字节数/SHA-256 与本机构建一致；当前 20901 及更早已检查版本可取得 21000 更新，21000/21001 不重复更新或降级。公开 Web 配置为 2.0.10，HTML 及引用的 JS/CSS/图标与发布包一致，HTML `no-cache`。
- 026 迁移由 root Unix socket 维护连接执行；21 份原迁移摘要保持一致，新增后共 22 条迁移。历史 **1 条全额退款券占用已释放**。该券当前仍为停用状态，因此不展示为玩家可用券；此次没有改变其启停或有效期。
- 数据库保留 **26 笔订单、4 个委托、70 个账号**，商品/店铺/券状态及既有 Saki 售卖配置保持。只读会话检查 24 笔当前账号可见订单及详情、钱包、商品、优惠接口；可见数量与数据库总量的差异不表示删除记录。
- 实际服务进程执行新 Go 路径，健康检查 ready；`config.json` 与 `backend.env` 摘要不变。验证临时会话已撤销并确认 401。
- Login、Amiya、Odyssey 在线，MEK 离线。Mail API 3、commerceReady、creditRewards 能力正常；Core/Mail/Bridge/XConomy/Sync 未更换，游戏服未重启。
- 更新通知只发给 `xiwanzi`，相同事件键仅 1 条，未广播公共聊天。没有创建生产测试券、订单或删除用户商品。

## 产物及备份

| 产物 | 字节数 | SHA-256 |
| --- | ---: | --- |
| App 21000 | 23557745 | `22a99ec7a1fdcf0182f124c8d9ba70d1c4ccc3953340f4d1755a22fbe26c65d5` |
| Go Linux amd64 | 18501935 | `1fa7b3dc691bc80cf7a579be832cec1465724225b8afbfb845b66b03d2666ee3` |
| Web 2.0.10 ZIP | 33697130 | `f4113bbc987c6bac1b270719a114ecae24d552c38cd4161c7f75958aa7ae6db1` |

App 包名 `com.deuterium.app.uilab`，与 20901 同签名，证书 SHA-256 `04a213066a78a968c2a1328a153a0967f99d117e0a826285e3f4af705db6f2f3`。

Go：`/opt/deuterium/releases/release-2.0.10-65c08a290896/deuterium`；Web：`/var/www/deuterium/releases/web-2.0.10-65c08a290896/dist`。

私有备份：`/var/backups/deuterium/release-2.0.10-20260911-65c08a290896`，含配置、环境、原更新清单、systemd/Nginx 配置、原服务指向与在线/停服 SQL 快照。02:38:36 停服快照 **1,500,833 字节**，SHA-256 `d255850755d4d61e1a5068f6d4019f1a5e10bf6935911b0560ce648e8bb2e7db`；备份压缩完整性已检查。确认无进行中资金操作后切换，原运行账号未扩大 DDL 权限。

回退必须保留删除终态、返券资格、原订单及后续新交易，不能恢复整个旧数据库或回填已被再次使用的旧券占用。旧 Go 不认识新删除终态，必要时使用保留这些规则的兼容修正版；已安装 21000 的设备只能使用更高版本码修正。

[构建清单](artifacts/v210/build-manifest.json) · [备份](artifacts/v210/cloud-prepared.json) · [停服快照](artifacts/v210/cloud-paused.json) · [发布](artifacts/v210/published.json) · [公网](artifacts/v210/public-verification.json) · [真实接口](artifacts/v210/backend-readonly.json) · [退券迁移](artifacts/v210/coupon-migration-verification.json) · [账号提醒](artifacts/v210/update-notice.json) · [最终状态](artifacts/v210/final-state.json)
