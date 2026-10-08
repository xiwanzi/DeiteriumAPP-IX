# 大目录商城修复验收（2026-10-08）

2026-10-08 后续状态：本候选已随 App 2.0.19 / Web 2.0.28 上线，见[发布记录](../deployment/deuterium-2.0.19-live.md)。下方保留发布前候选的验证范围。

状态：本机候选，未推送、未合并、未部署。基线 `4590afe`，分支 `xiwanzi/catalog-scale`。[实施边界](../plans/catalog-scale-20261008.md)。没有修改生产商品、价格、库存、账号、订单或 OSS 权限。

## 原因与修复

1. **网页全部售罄**：线上只读核实 3,085 件在售商品均为不限库存，封面绑定缺失数为 0。网页详情原来将 `availableStock=null` 回退到 `content.stock=0`。现在按 `inventoryPolicy=UNLIMITED` 处理，保留单笔数量上限，有限库存的实时 0 仍禁购。
2. **首屏等待全量目录**：App 原来 `listAll` 连续取完每页 100 件的全部商品后才提交 UI。现在首屏 40 件，距离已读末尾 12 件时取下一页；并行读取品牌/分类。浏览与搜索分别持有游标，快速切换时旧请求不能覆盖新结果。购物袋独立读取关联商品，网络失败不会误判为下架或用缺失的部分商品结算。
3. **离开页面红字**：用户确认提示接近 `The coroutine scope left the composition`；已在实际 Compose 1.10.3 依赖的 `LeftCompositionCancellationException` 中核实原文。旧 `runCatching.onFailure` 将正常离开页面取消写入 `shopError`。新分页器传播取消且不写入错误；真实续页失败保留商品，重试按钮放在列表末尾。
4. **滚到商品才请求图片**：首图向下预取最多 24 件（搜索 16 件），后台只占 1 个下载槽，保留可见图片的带宽。预取直接写现有受大小约束的磁盘缓存，不解码占用图片内存；签名轮换仍按资产 ID 复用字节。授权 URL 缓存从 1,024 条改为 8,192 条、按最近使用淘汰；过期链接刷新对应商品，不重取整页。
5. **后台逐图 SQL**：公开商品列表批量读取精确的商品/图片绑定和元数据，100 图由 200 次 SQL 降为 2 次。绑定缺失仅该图降级，不隐去整页；数据库错误仍报错。发布快照、价格和鉴权边界保持，搜索补上分类名称匹配。无需迁移数据库，也无需改变旧客户端 API。

## 本轮证据

| 项目 | 结果 |
| --- | --- |
| Web 单元测试 / 生产构建 | 74 项通过；Vite 构建通过，保留既有大 chunk 提示 |
| 真实 Chrome，3,085 件隔离目录 | 首次仅 1 次商品请求、40 张卡片；不限库存详情可购买；搜索末项；慢旧查询不覆盖新查询；断页保留 40 件并重试到 80 件；390px 宽无横向溢出 |
| Go 普通测试 | `go test ./...` 通过 |
| Go 隔离数据库 | 14 项 Catalog 测试通过；补充分类搜索的大目录用例通过；发布草稿/头像图片绑定与权限回归通过 |
| 100 图渲染对比 | 两次本机结果：旧 139 / 199 ms，批量 26 / 31 ms；SQL 始终为 200 → 2。包括 3,085 件的首个 40 件 HTTP 页分别约 12 / 31 ms。这是本机隔离数据库结果，不是公网或手机实测时延 |
| Android JVM | 160 项通过，包括分页失败重试、旧搜索隔离、取消、重复游标、3,100 条图片授权及 LRU 边界 |
| Android 构建 / lint | Debug、测试 APK 及启用 R8/资源压缩的 performance APK 构建通过；lint 0 errors、20 warnings。用户确认的取消文案另以明确的 CancellationException 补跑 6 项分页用例，通过 |
| Android 35 原生验收 | 实际商城组件首屏仅 40 件；滚动提前续页、503 后重试；搜索第 3,085 件；购物袋独立补齐未读过的商品。响应体延迟 150ms 时，自动续页从 80 到 120 件且仅发出一次请求，不因 loading 状态重组被取消。Google APIs 模拟器复验和浅深色截图完成 |
| 图片预取 | 真实 HTTPS 测试图片预取后，首次显示来自磁盘；预取未新增内存位图。已有磁盘/内存复用、签名更新、过期限制、私有证据不缓存和清缓存检查通过 |

浏览器隔离入口：`web-app/qa/catalog-scale.html`，使用真实 `RemoteCollection` 和商品详情组件。该入口不在生产构建依赖中。Android 图像网络试验使用官网既有静态 PNG，只执行 GET；未修改 OSS 对象。

本机截图、请求序列和浏览器 JSON 在 `C:/DeuteriumAPP/deliverables/catalog-scale/`；构建与测试日志在 `C:/DeuteriumAPP/.tools/catalog-scale-*.log`。ATD 功能检查通过，但 ATD 的黑色截图不作为视觉证据；已用 Google APIs 模拟器重新截图。

## 发布边界

候选沿用 App 2.0.18（21800）和 Web 2.0.27 的版本号，不能当作新版本在线交付。正式发布前需按新版本编译和发布清单，依本轮发布授权执行。商品分页与预取兼容旧后端；完整优化须同时替换 Web、Go 和 App。

本机候选 `C:/DeuteriumAPP/deliverables/catalog-scale/Deuterium-2.0.18-21800-catalog-scale-candidate.apk`，8,908,306 字节，SHA-256 `451bd05ec7b2d9de22e8fcaf69543458984231da9485d6832ff300420315805e`。包名 `com.deuterium.app.uilab`，证书 SHA-256 `04a213066a78a968c2a1328a153a0967f99d117e0a826285e3f4af705db6f2f3`，与原发布包一致；沿用原版本号，仅用于候选验证。原生功能/截图验收使用同源 Debug APK，performance 包完成编译及签名/元数据校验，未将两者混写为压缩包完整原生回归。

构建命令：`./gradlew.bat :ui-lab:assembleDebugAndroidTest :ui-lab:assemblePerformance :ui-lab:testDebugUnitTest :ui-lab:lintDebug --offline --console=plain`。Linux amd64 Go 候选已用 `go build -trimpath -ldflags='-s -w' ./cmd/deuterium` 编译到同一交付目录；哈希及检查摘要见该目录 `build-manifest.json`。

尚无用户真机的公网首屏/弱网滚动时延；模拟器和本机基准不替代该验收。现有页面布局保留，大目录的整体 UI 改版不在本轮范围。
