# 商城布局演示 1.1 验收

2026-10-08。用户明确授权参考京东重新设计商城卡片与首屏分配，见[本轮计划](../plans/shop-commerce-layout-20261008.md)。本机独立离线演示；正式 App、Web、Go 与业务数据库未发布或修改。

## 已完成

- 常驻搜索缩至 44 dp，分类改为文字频道与选中短线，右侧固定更多。
- 活动头图 108 dp 并展示已配置标题；新品改为 164 dp 宽、112 dp 图片区的紧凑横滑卡。首屏可见普通双列商品，首页下方去除与当前新品横滑区重复的商品；分类/搜索仍展示匹配的全部商品。
- 首页、一级分类页、搜索使用双列瀑布流，卡片独立向下排布。左右边距/列间距/行间距均 8 dp，圆角 12 dp；图片、名称、分类、蓝色价格、信用点单位和直接加购有明确层次。标签与折扣仅按实际数据展示。
- 统一背景商品图按可见主体边界等比绘制；保留安全边，细长物体不横向拉伸。复杂照片、空图保留完整图，失败降级原图片组件。原始素材文件不变，不新增放大后的图片副本或独立磁盘缓存。
- 分类网格图按格宽自适应，修正原 52 dp 图框叠加素材留白造成的商品过小。
- `DeuteriumActivity.kt`、`MovingGlassNav.kt`、`LiquidGlass.kt`、`GlassParameters.kt` 本轮无变化；原版底栏、玻璃与全局动效保留。

## 验证

- 构建 `:ui-lab:assembleShopDemo`、`testDebugUnitTest`、`lintShopDemo` 通过，均使用 `--offline --console=plain`。163 项 JVM 用例通过，0 失败；lint 0 errors、14 warnings。
- 新增主体边界用例覆盖细长物体、白底浅色物体、透明底黑色物体、纯色空图与复杂场景，确保不切掉主体或凭空裁图。
- 安装最终演示 APK 后验证首屏商品流可见、卡片内加购、本地购物袋、继续浏览第 40 件以后的商品、详细分类、关键词搜索与浅深色。深色截图等待图片解码稳定后重新检查。
- 包名仍为 `com.deuterium.app.shopdemo`，版本 `1.1-demo` / `2`，签名与演示 1.0 相同；最终 APK 不含 `INTERNET` 权限。可覆盖上一演示版而不影响正式 App。
- 此轮 ADB 只检测到模拟器，物理手机断开；尚未安装到用户手机，不将模拟器验收当作真机触感验证。

## 交付

目录 `C:/DeuteriumAPP/deliverables/shop-demo-1.1/`：

- `Deuterium-Shop-Demo-1.1.apk`：14,905,386 字节。
- SHA-256：`ae77a13cf819dd04a02e8bae56b9534c052ddb56dad263dab4bee28359f2128c`。
- `home.png`、`feed.png`、`categories.png`、`search.png`、`home-dark.png` 为实际安装包截图。
- `verification.json`、`apk-metadata.json`、`build-manifest.json` 为操作和包信息核验。

演示目录仍为内置 219 件商品；原版 1.0 文件保留。后台商品展示预览尚未同步此次本机设计，正式发布前需要另行同步，不能据此宣称正式业务已经采用新版。
