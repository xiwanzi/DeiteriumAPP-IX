# 商城导航与邮件标题本机验收

2026-10-08，基线 `5d7523d`，工作区 `.worktrees/catalog-scale`，分支 `xiwanzi/shop-navigation`。**本机候选，未推送、未合并、未部署。** [实施边界](../plans/shop-navigation-20261008.md)。

## 视觉返工与用户纠正

初次验收容器错误地用文字底栏替代了正式底栏，并用横向场景图代替商品图，不能证明遵守原视觉规范。用户指出后撤回这组截图的验收结论；同时恢复本次擅自更改的搜索框与双列卡外观。

- 正式 `DeuteriumActivity` 底栏调用与原版逐字一致，`MovingGlassNav.kt`、`LiquidGlass.kt`、`GlassParameters.kt` 无改动。验收现在使用同一 `LiquidGlass`、图标、指示器、70 dp 高度、30 dp 圆角和既有玻璃参数，并采用真实 `GlassScene` 采样层。
- 固定搜索栏沿用原来的 52 dp 高度、17 dp 圆角、半透明底及原字号；只改变位置和常驻行为。
- 双列卡恢复原来的 24 dp 圆角、170 dp 图片高度、16 dp 内边距、20 dp 页面边距及 22 dp 行距，移除未经要求新增的品牌行和卡片标签行。
- 新品海报继续使用原 `ProductPoster`，函数体与原版一致，尺寸仍为 308 × 405 dp，图片为等比 cover；使用实际导入商品（蓝蘑菇床等）原始 384 × 384 图复查，不使用不相关场景图充当商品。
- 修正后的原生导航流程再次通过。有效截图位于交付目录的 `reviewed/`；原先未带原版导航的截图不作为视觉交付。

## 已实现

- 游戏邮件默认标题改为“官方商城订单”；空自定义标题和多商品合单均不再拼订单号。单商品已配置的自定义文案继续生效，正文、订单关联与既有邮件快照保持。后台输入提示与预览同步。没有修改 Core、桥接或 Mail 模组。
- 商城不显示大标题。固定搜索栏、购物袋、首页与可横滑一级分类、右侧更多；顶部状态栏下方保留 8 dp 点击区回到顶部，关闭灵动视效时直接回顶。其他主页面继续原折叠方式。
- 首页活动横图 → 带“新品推荐”标记的横向大卡 → 双列分页商品流。推荐“查看更多”精确筛选该标签。没有活动图或推荐标签时不伪造内容。
- 一级分类显示最多 15 个子分类；多于 15 个时显示前 14 个加更多。目录左侧一级分类、右侧子分类网格，子分类点击后进入独立双列搜索。搜索由搜索按钮/键盘提交，支持全目录查询与续页，返回保持原浏览状态。
- 分类图标按每类一个已发布商品封面批量读取，沿用精确图片绑定校验；不读取全量商品。跨店同名一级分类合并展示，`categoryGroup` 查询覆盖各店对应大类及子类。
- 商家后台新增分类层级、首页活动图入口，商品编辑新增“新品推荐”选项。活动图保留其他首页 section，推荐开关保留已有其他 badges。

## 用户确认的一级分类与整理方案

| 一级分类 | 现有细分类 |
| --- | --- |
| 家具家居 | 座椅沙发、收纳置物、桌台家具、床与寝具、厨房用品、卫浴用品、实验室陈设 |
| 灯具照明 | 室内灯饰、吊灯与壁灯、灯带与照明方块、台灯与落地灯、造型夜灯、庭院石灯 |
| 绿植庭院 | 绿植园艺、花卉植物 |
| 建筑建材 | 墙地建材 |
| 装饰摆件 | 装饰摆件、风铃、玩偶抱枕 |
| 其他 | 枪支 |

已只读核对现有 20 个细分类。整理工具为 [shop_navigation_categories.py](../../tools/shop_navigation_categories.py)，默认仅生成计划，必须显式 `--apply` 才通过现有商家 API 保存；从环境变量读取管理员 token，不输出或保存 token。版本与幂等沿用服务端。只创建一级分类及更新原分类归属，不改变商品 ID、分类 ID 或商品引用；同名现有一级节点复用。当前仅生成离线计划，尚未执行线上归并。

发布时先保留原分类配置备份，再替换候选 Go/Web、应用归并计划；通过后台选定新品推荐商品并上传实际活动图，最后交付对应 App。当前没有替用户选择实际推荐商品或上传测试活动图。分类字段为可选 JSON 扩展，无数据库迁移；回退归属使用备份配置和当时最新版本，不回滚整个数据库覆盖新增订单。

## 本轮验证

| 范围 | 结果 |
| --- | --- |
| Go 全包测试 | `go test ./...` 通过 |
| Go 本机隔离数据库 | `TestShopNavigationHierarchyBadgesAndMedia` 与原大目录回归通过；验证父分类商品、父分类名搜索、标签发布快照、精确标签、拒绝环/三级/跨店归属、分类封面失去绑定不泄露 URL、活动图权限 |
| 大目录批量图片回归 | 3,085 件隔离商品；原 100 图 200 次 SQL、新批量 2 次 SQL，未破坏原分页优化 |
| Android JVM | 160 项通过，0 失败/跳过 |
| Android 原生导航 | Google APIs Android 35；首页推荐、一级分类、超过 15 个的更多、短分类不显示更多、左右分类目录、子分类搜索、详情返回、连续分页、固定顶栏、回顶、浅深色、搜索第 3,085 件商品通过 |
| Android 原生分页回归 | 首屏仅 40 件普通商品、失败保留已读项、重试续页、延迟请求不重复/取消、全目录搜索、购物袋补齐未浏览商品全部通过；推荐为独立请求 |
| Android 构建 | Debug、测试 APK、performance（R8/资源压缩）通过；完整 lint 检查 0 errors、20 既有 warnings。最终回顶动作补充遵守动效开关后再次构建 performance 成功 |
| Web | 74 项测试及生产构建通过 |
| 真实 Chrome | 分类保存带 expectedVersion/父 ID；修改 banner 保留 intro、品牌和原商品区；推荐勾选保留既有“手作”标签；默认邮件短标题；390 px 无横向溢出，0 pageerror |

原生导航验收为真实生产 Compose 页面、原版导航/玻璃组件和隔离 HTTP 数据。返工后的商品图来自真实导入源；商品条目与推荐标记仍为本机夹具，活动图也只是布局示例，没有发布为实际运营内容。原生回归运行 Debug，同源 performance 完成编译与签名校验，没有冒充压缩包已完成所有原生流程。尚未在真实游戏客户端复测 tips，也未进行用户真机公网手感验收。

## 本机交付

目录：`C:/DeuteriumAPP/deliverables/shop-navigation/`。

- `Deuterium-2.0.19-21900-shop-navigation-candidate.apk`：包名 `com.deuterium.app.uilab`，版本仍为 **2.0.19 / 21900**，候选不进入在线更新。证书 SHA-256 `04a213066a78a968c2a1328a153a0967f99d117e0a826285e3f4af705db6f2f3`，与当前线上一致。
- `deuterium-shop-navigation-linux-amd64`、`web-shop-navigation-candidate.zip`：配套候选。
- `build-manifest.json`：最终文件大小、SHA-256 与测试摘要。
- `reviewed/shop-home-light.png`、`reviewed/shop-home-dark.png`、`reviewed/shop-furniture.png`、`reviewed/shop-categories.png`、`reviewed/shop-category-search.png`、`reviewed/shop-feed.png`、`reviewed/shop-keyword-search.png`：返工后的本机布局证据。
- `web-verification.json`、`web-mail-preview.png`、`web-settings.png`：后台验证。

构建命令：`./gradlew.bat :ui-lab:assembleDebug :ui-lab:assembleDebugAndroidTest :ui-lab:assemblePerformance :ui-lab:testDebugUnitTest :ui-lab:lintPerformance --offline --console=plain`；最终小改动单独重建 `:ui-lab:assemblePerformance`。Go：`GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' ./cmd/deuterium`。网页：`npm test`、`npm run build`。

只安装 APK 连接旧 Go 时，尚未部署的分类归并、标签搜索及活动图扩展不会完整生效；不能把本机截图当作线上状态。旧游戏邮件不会因新默认规则自动改名。
