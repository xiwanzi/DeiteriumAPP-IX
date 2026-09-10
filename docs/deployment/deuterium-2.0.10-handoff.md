# Deuterium 2.0.10 测试交付

2026-09-11。用户已授权合并与推送源码，并准备手机测试。

优惠券应用内通知、券/商品删除、全额退款返券和启动绘制修正已通过 [PR #18](https://github.com/xiwanzi/DeiteriumAPP-IX/pull/18) 合入 main，合并提交 `303db4ee57f95c967e666b029ce99b5b9bfd52e7`。合并后的 Android/Web/Go 业务目录与已验收的 `48da9a3` 一致，无冲突、无新增业务修改。

App/Web 版本调整为 **2.0.10**，App **21000**，用于区分线上 2.0.9（20901）并支持覆盖安装。Web 运行配置、包元数据、版本断言和启动日志同步。

## 测试包

本机交付路径：`C:/DeuteriumAPP/delivery/Deuterium-2.0.10/Deuterium-2.0.10-21000-test.apk`。

| 项目 | 值 |
| --- | --- |
| 包名 | `com.deuterium.app.uilab` |
| 版本 | `2.0.10 / 21000` |
| 大小 | 23,557,745 字节 |
| APK SHA-256 | `22a99ec7a1fdcf0182f124c8d9ba70d1c4ccc3953340f4d1755a22fbe26c65d5` |
| 证书 SHA-256 | `04a213066a78a968c2a1328a153a0967f99d117e0a826285e3f4af705db6f2f3`，与线上 20901 一致 |

本次版本提升后重新运行 `gradlew.bat :ui-lab:testDebugUnitTest :ui-lab:assembleDebug --console=plain`，134 项单测和 APK 构建通过；APK 元数据、完整性及签名复核通过。Web 65 项测试及生产构建通过。Go 未改业务源码，沿用本轮开发阶段已通过的全量竞态/集成结果，没有将其写成此次重新运行。

## 当前部署边界

**此记录只证明源码合并、推送及本机测试包准备，不代表线上发布。** 线上 App/Web/Go 仍按 [2.0.9 记录](deuterium-2.0.9-live.md) 运行；未发送账号更新提醒。

安装本包可先测试优惠券顶部通知和启动/加载体验。完整线上测试商品/券删除和退款返券，需要同步部署 Web、Go，并在备份后执行 `026_refunded_coupon_release.sql`。该部署范围已向用户单独核对，等待明确答复；不在未确认时运行生产迁移。

不需要更换 Core、Mail、Bridge、XConomy 或重启游戏服。发布时保留现网配置、订单、委托及运行状态；026 只释放已确认全额退款订单对应的券占用，不能回填旧占用覆盖后续交易。

[功能验收与截图](../qa/commerce-refunds-performance.md) · [性能审查及真机验证边界](../reviews/startup-performance-2026-09-11.md)
