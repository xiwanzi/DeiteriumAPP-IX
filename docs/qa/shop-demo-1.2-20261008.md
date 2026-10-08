# 商城离线演示 1.2

2026-10-08，按用户本轮截图和要求修改，覆盖演示 1.1 的首页区块安排。

## 本轮效果

- 首页为固定搜索/频道栏 → 原版 308 × 405 dp 商品大卡横滑 → 双列信息流。移除活动 banner、独立新品小卡区、“新品推荐”“逛逛好物”等区块标题。
- 大卡取当前目录前六件商品，信息流与大卡去重；分类和搜索仍能找到这些商品。首页不再请求活动图或新品标签目录，不修改后台已有字段或标签记录。
- 商品卡按参考改为方形图片、8 dp 圆角、约 7 dp 间隔；名称单行、分类说明紧凑，价格和信用点同一行，加购图形缩小但保留按钮。长金额按可用宽度缩放，未截断金额。
- 分类页左栏 104 dp；右侧增加横向分组导航、可展开分组入口、圆角白色内容面和连续分组标题，分类图框最多 72 dp。
- 家具家居按“起居家具 / 收纳陈设 / 厨房卫浴”、灯具按“室内 / 庭院”做展示分组；它们不是新增数据库层级。点击真实子分类仍搜索原分类名，原 ID 与商品归属不变。
- 原底部导航、玻璃、全局动效和原始商品图片未改。

## 验证与边界

`./gradlew.bat :ui-lab:assembleShopDemo :ui-lab:lintShopDemo :ui-lab:testDebugUnitTest --offline --console=plain` 通过。163 项 JVM 用例，0 失败；lint 0 errors、14 warnings。

Android 35 实际安装包验证：旧 banner 和两段标题不出现、原大卡可进入详情、卡内加购、本地购物袋、超过首 40 件继续分页、分类分组选择/展开、子分类关键词搜索通过。截图为实际安装包输出。

独立包 `com.deuterium.app.shopdemo`，`1.2-demo` / versionCode `3`，无 INTERNET 权限，可覆盖演示 1.0/1.1而不覆盖正式 App。原演示目录和数据继续内置；没有发布正式 App/Web/Go，也没有写入业务数据库。ADB 当前仅模拟器在线，物理手机尚未安装本版。

## 文件

目录 `C:/DeuteriumAPP/deliverables/shop-demo-1.2/`：

- `Deuterium-Shop-Demo-1.2.apk`，14,905,386 字节。
- SHA-256：`27155e67a9fd9e85a5fa375167f4c42596c95aca8732edc8baa184763e98ca22`。
- `home.png`、`feed.png`、`categories.png`、`category-selected.png`、`search.png`。
- `verification.json`、`apk-metadata.json`、`build-manifest.json`。

手机实际观感与触感以安装后体验为准。后台商品预览尚未同步本轮设计，正式发布前另行处理。
