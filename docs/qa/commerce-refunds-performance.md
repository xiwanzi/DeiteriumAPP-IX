# 优惠券、商品删除及性能修正验收

2026-09-11。本机分支 `xiwanzi/commerce-refunds-performance`，从 `e5ea63e`（线上 2.0.9 部署记录）建立独立工作区。**尚未推送、合并、部署或发送更新通知**。当前公网仍按原 2.0.9 发布记录维护。

## 本机结果

| 范围 | 本轮验证 |
| --- | --- |
| Go | `go test -race -tags integration ./...` 全量通过；HTTP 包 317.626 秒，独立集成包 108.192 秒。使用本轮启动的回环 MariaDB 和测试框架随机建库，未连接生产数据库 |
| 删除 | 非管理员不能删券；启用券须先停用；草稿可以删除。商品删除校验店铺及发布权限、版本与重复请求；删除和购买并发仅一方接受同一版本；删除后的旧报价、编辑及发布被拒绝 |
| 原订单 | 删除在售商品后，原订单快照可读，退款成功，库存可恢复且商品仍为 `DELETED`；经营列表不再展示该商品 |
| 退券 | 全额退款结果未知时继续占用，证明确认后恢复资格；旧退款重放与重复迁移不影响新订单重用。过期、停用、删除券不会恢复成可用券；已查看券不会重新触发到账提醒；失败退款保留占用；补充运行全额抵扣 0 元订单返券断言通过 |
| 历史修复 | 026 迁移只清除已确认全额退款订单的原占用；未知结果、待处理操作、部分退款或非零结算均保留。重复执行不损失订单及批次历史 |
| Web | 65 项测试及 `npm run build` 通过。Playwright + 隔离真实 HTTP/数据库验证：停用→出现删除入口→确认删除；断网失败后原请求重试成功，列表移除；在售商品删除后经营列表数量由 2 变 1 |
| Android | 134 项单测通过（移除 1 项旧弹窗按钮规则测试）；APK/测试 APK 构建成功、同签名覆盖安装成功。Android 35 x86_64 原生交互脚本 7 组通过 |
| 原生提醒 | 单券/多券统一顶部通知；浅深色、大字、关闭玻璃/动效、支付浮层避让；卡片外页面可点击，卡片可进入优惠页；离线重建、同批稍后生效、账号隔离、到期消失与红点验证通过 |
| 启动 | 桌面别名冷启动成功，徽标和加载条正常绘制。真实性能收益不以单次模拟器耗时代替，见[性能审查](../reviews/startup-performance-2026-09-11.md) |

网页夹具中的商品图片指向测试对象存储，预览可能返回 404；验收只将真实接口/数据库、删除交互和布局作为证据，没有把夹具当成生产内容。原生到期断言已按具体券文案等待，避免误把上一条通知的退出动画视为下一条券到期。

## 本机构建

APK：`android-app/ui-lab/build/outputs/apk/debug/ui-lab-debug.apk`，23,557,749 字节，包名 `com.deuterium.app.uilab`，版本 `2.0.9 / 20901`。本轮未申请正式发布，因此未提高在线版本码；这是开发验证包，不是新的在线更新版本。

- APK SHA-256：`9b7c7d3999f498e9219e6ea8ae4c6dc3f244eaa3cfe784fafff0bc772eb1ea50`
- 签名证书 SHA-256：`04a213066a78a968c2a1328a153a0967f99d117e0a826285e3f4af705db6f2f3`，与线上 20901 一致。
- 构建命令：在 `android-app` 执行 `gradlew.bat :ui-lab:testDebugUnitTest :ui-lab:assembleDebug :ui-lab:assembleDebugAndroidTest --console=plain`。
- 新后端发布需先备份、执行 `026_refunded_coupon_release.sql`；回退须保留删除终态和返券资格语义，不能回填旧占用覆盖新交易。详见[契约](../contracts/commerce-promotions-v209.md)。

## 截图

[应用内通知（浅色）](artifacts/commerce-refunds/coupon-simple-light.png) · [深色](artifacts/commerce-refunds/coupon-simple-dark.png) · [大字](artifacts/commerce-refunds/coupon-simple-large-text.png) · [多券](artifacts/commerce-refunds/coupon-multiple-light.png) · [券删除（手机）](artifacts/commerce-refunds/coupon-delete-mobile.png) · [商品删除（桌面）](artifacts/commerce-refunds/product-delete-desktop.png) · [开屏](artifacts/commerce-refunds/startup.png)

未完成的验证范围：用户骁龙 8E 真机帧轨迹、正式构建性能基准和真实手机手感。游戏插件协议无需改变；本轮没有操作实服订单、优惠券或商品。
