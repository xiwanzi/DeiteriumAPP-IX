# App 2.0.19 / Web 2.0.28 大目录商城修复上线

2026-10-08 **14:31（UTC+8）**，App **2.0.19（21900）**、Web **2.0.28** 与配套 Go 已上线。[PR #79](https://github.com/xiwanzi/DeiteriumAPP-IX/pull/79)，源码 `42ea3dcd05a140c06086339ba04a1fc5f7a12c9a`，合并 `50c452b8b845f54c0fbdb4142a234963238a5ac3`。构建源码与合并源码的 Android、Web、Go 树一致。

## 已生效

- 网页不再将不限库存商品的 `null` 库存回退为 0；有限库存的实时 0 仍禁购，成交仍由后端报价校验。
- App 首屏读取 40 件商品，接近页尾时提前续页；搜索查询完整目录，购物袋独立补齐未浏览的商品。
- 离开页面的 `The coroutine scope left the composition` 取消不再误报红字。续页失败保留已读商品，并在列表末尾提供重试。
- 预取后续最多 24 件商品首图，复用原磁盘缓存，授权 URL 缓存按最近使用淘汰；过期链接刷新对应商品。
- Go 按页批量读取授权图片。100 图的图片元数据/绑定查询由 200 次降为 2 次；单张图片缺失不阻断整页。

现有整体布局保持，后续大目录 UI 改版另行讨论。官网、客服中心、游戏插件及配置保持，本次没有运行数据库迁移。

## 发布和安装

[下载 App 2.0.19](https://47.103.99.34/downloads/Deuterium-2.0.19-21900-test.apk)。包名 `com.deuterium.app.uilab`，APK 8,908,306 字节，SHA-256 `eabda7c59073e0d495ea0a94080bc55d6db478f5231ad704cc25737d11a86802`。证书 SHA-256 `04a213066a78a968c2a1328a153a0967f99d117e0a826285e3f4af705db6f2f3`，与原发布包相同。

App 内检查更新可获取新版。原 19 个发布条目完整保留；20004、21300、21500、21700、21800、21899 均返回 21900，21900 和 21901 无重复更新或降级。未另发账号消息。

正式包使用 `performance` 变体，非 debuggable，启用 R8、资源压缩与 Baseline Profile。构建命令：`./gradlew.bat :ui-lab:assemblePerformance :ui-lab:testDebugUnitTest :ui-lab:lintPerformance --offline --console=plain`。Android 35 Google APIs 模拟器覆盖安装后确认版本 21900 / 2.0.19，经 MainActivity 别名启动正常。

## 验证与数据

- 本轮正式版本构建完成，160 项 Android JVM、74 项 Web 测试通过；本次功能候选另完成 Go 全包测试、14 项目录集成测试、图片权限回归、Chrome 与 Android 原生验收，见[详细验收](../qa/catalog-scale-20261008.md)。未将候选验收冒充正式压缩包重新完成的全量原生回归。
- 两个公网入口 `47.103.99.34`、`chat.deuteriumix.com` 的 Web 配置均为 2.0.28；实际 JS/CSS 与本机构建逐字节一致。公网 APK 的完整大小和 SHA-256 一致。
- 实际商城首、次页各 40 件，无重复；不限库存、详情和搜索返回正确；封面从 OSS 下载后 SHA-256 一致。本机至公网 HTTPS 的单次首/次页为 **305.7 / 228.3 ms**，搜索为 **396.0 ms**；这不是用户手机首屏或弱网体验数据。
- Go ready / active，Login、Amiya、Odyssey、MEK 四节点全部在线。用于只读验收的短期管理员会话已撤销。
- 切换前后均为 **56 条账号记录、38 笔订单、5 个委托、3,085 件在售商品、12 个工单、2 条注销记录**。额外比对账号、交易、目录及图片的既有 ID 集合，均完整保留。
- schema、业务配置与环境文件哈希保持；没有迁移或清理业务数据，没有恢复旧账号或旧白名单。

真机公网首屏及连续滚动手感仍待用户安装后复测。

## 备份与回退

备份目录 `/var/backups/deuterium/catalog-scale-21900-42ea3dcd05a1`，含一致性数据库备份、旧程序、发布清单与配置备份。新 Go 位于 `/opt/deuterium/releases/catalog-scale-42ea3dcd05a1`，Web 位于 `/var/www/deuterium/releases/web-2.0.28-42ea3dcd05a1/dist`。

若需回退，恢复原 Go/Web current 链接与旧清单后重启原 Go；无数据库迁移，不恢复数据库覆盖上线后新增业务。原 Go 为 `auth-flow-0248496b034a`，原 Web 为 `web-2.0.27-0248496b034a/dist`。

本机交付 `C:/DeuteriumAPP/delivery/Deuterium-2.0.19/`，含正式 APK、R8 mapping、Go、Web 压缩包和验证记录。

[构建清单](artifacts/v219/build-manifest.json) · [切换记录](artifacts/v219/deployment.json) · [公网与节点验证](artifacts/v219/verification.json)
