# 独立离线商城演示 APK

2026-10-08，按用户要求单独生成演示包：不连接现有业务数据库，不使用正式 App 包名。

## 交付

- 名称：Deuterium 商城演示。
- 包名：`com.deuterium.app.shopdemo`，可与 `com.deuterium.app.uilab` 共存。
- 版本：`1.0-demo`，versionCode `1`。
- 文件：`C:/DeuteriumAPP/deliverables/shop-demo/Deuterium-Shop-Demo-1.0.apk`。
- 大小：14,889,006 字节；SHA-256：`904387167dd6aa55f69760ceb3a10b2324782e5cf1a182f8793e51a08fbbc2a7`。
- 签名校验通过，已在 Android 35 模拟器安装、冷启动；正式 App 同时保留。未发布到正式在线更新。

## 隔离与界面

独立 `shopDemo` build type/source set 内置 219 件演示商品、分类及原始图片。数据只取公开商品展示资料，ID 为本地演示 ID，不含真实账号、订单、登录凭据或数据库配置。

最终 APK 没有 `INTERNET` 或 `ACCESS_NETWORK_STATE` 权限；删除更新安装权限、正式入口、更新接收器和文件提供者。保留共用 UI 所需的通知权限声明，但没有实时连接或通知事件来源。

`OfflineShop` 拦截器仅在内存/本包偏好存储中处理目录、搜索和购物袋，绝不调用 `chain.proceed`。未知业务操作明确返回“离线演示，不执行真实交易”；付款/建单不可用。图片从内置资产写入本包缓存，基址为不可用的 `https://offline.invalid`。

演示入口直接复用正式 `LabApp` 与所有商城页面；通过可选 `providedState` 注入本地状态，跳过 `LabSession` 的联网初始化。正式入口默认参数不变。底部导航、玻璃、动效和界面组件不另造替代品。主题与本地购物袋仅保存在演示包自己的沙盒内。

## 构建与验收

构建：`./gradlew.bat :ui-lab:assembleShopDemo :ui-lab:lintShopDemo --offline --console=plain`。JDK 17、现有 Android SDK；没有安装新的构建依赖。

最终 APK 经 aapt 核对包名、版本、入口和权限；apksigner 校验通过。真实安装包走通首页、详情、加购、本地购物袋、分类目录及关键词双列搜索；确认离线提示与真实交易禁用。截图和校验 JSON 在同一交付目录。正式版的 160 项 JVM 回归检查另运行，结果见构建日志。

此包用于商城体验；其他业务没有接入演示后台，也不会连接正式服务。Android 模拟器不代替用户手机的流畅度和触感验收。
