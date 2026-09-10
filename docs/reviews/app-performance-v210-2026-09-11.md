# App 2.0.10 启动收退与首页首次滑动性能审查

> 后续实施已记录于 [性能优化验收](../qa/app-performance-equivalence-v210.md)。下文“仅审查、无连接设备”描述本报告形成时的状态；不代表后续实施的验证状态。

2026-09-11。审查当前发布 App **2.0.10（21000）**，重点对应“启动图标往里退时顿挫”和“刚开始滑首页卡顿”。

**后续约束（同日）：用户要求扩展到其他页面，并严格保证效果实现完全一样。** 本文建议已据此收紧；完整范围、允许的等效优化和验收条件见 [全 App 性能复核](app-performance-full-v210-2026-09-11.md)。不能以降低视觉质量、替换动画、改变加载/交互行为来实现性能提升。

**结论：存在需要处理的性能问题，上一轮优化还没有覆盖启动交接和首页折叠链路。** 最直接的代码热点是首段滑动逐帧改变整个列表的顶部间距和标题排版；启动还有重复商城请求、批量主线程更新及徽标异步就绪缺口。发布包实际仍为 debug，且没有打包 Baseline Profile，会影响首次执行表现及后续性能判断。各项在用户手机上占用多少时间，尚无帧轨迹可以排序。

## 审查对象与本次证据

- 发布源码 `65c08a290896601c6254369792e4223216ebee8e`；审查工作区基于 `ad8a8ef`，已核对两者的 `android-app` **没有差异**。
- 本机 APK 为 23,557,745 字节，SHA-256 `22a99ec7a1fdcf0182f124c8d9ba70d1c4ccc3953340f4d1755a22fbe26c65d5`，与 [2.0.10 发布清单](../deployment/artifacts/v210/build-manifest.json) 相同。
- 本轮运行 Android 构建诊断、APK 元数据核验、SHA-256/ZIP 条目检查、源码调用链审查及 ADB 设备检查，原始结果见 [APK 审查证据](artifacts/app-performance-v210/apk-audit.json)。SDK 已定位；首次诊断没有环境变量，随后显式指定 SDK 完成核验。
- ADB 当前没有连接设备。**本轮没有真机或模拟器帧率结果**，也没有测得 CPU/GPU 耗时、掉帧率或加速比例。用户手机当前安装版本仍需设备核对。
- 本轮仅新增审查文档及证据，未修改 App 源码、提高版本、重新构建、推送或部署。历史 134 项单测和旧模拟器验收不计入本次验证。

## 发现与优先级

优先级表示建议处理顺序；“已确认”指代码或 APK 事实，不代表已用真机证明它是唯一卡顿原因。

| 优先级 | 发现 | 对应表现 | 证据强度 |
| --- | --- | --- | --- |
| P1 | 发布 debug 包，无打包的 Baseline Profile | 冷启动、首次进入与首次执行缺少正式构建优化条件 | APK 与配置已确认，实际增量耗时未测 |
| P1 | 首页折叠每帧改变列表间距、标题字号与布局 | 刚开始向上滑的 128dp 折叠区间更重 | 状态读取和调用链已确认 |
| P2 | 系统开屏交接未等异步徽标显示 | 收退动画可能从中途才被看见，形成视觉跳变 | 存在时序缺口；需延迟加载/真机录像确认发生频率 |
| P2 | 启动从两个入口刷新整套商城数据 | 首屏请求、映射和状态提交重复，可能与动画/首滑重叠 | 双调用入口与无去重实现已确认 |
| P2 | 就绪判断与首页内容无关，多类非当前页数据集中回到主线程 | 动画结束后首页还在填充，首次滑动与批量更新竞争 | 调度位置与逐项排序已确认，单帧耗时未测 |

### 1. P1：首先建立可用于性能验收的正式构建

[ui-lab/build.gradle.kts](../../android-app/ui-lab/build.gradle.kts) 第 7–31 行没有配置 release 的 R8 优化；发布 APK 的 `aapt dump badging` 明确包含 `application-debuggable`。ZIP 检查没有 `assets/dexopt/` 或 `.prof/.profm` 文件，工程也未配置应用自己的 Baseline Profile 生成模块。

这比“依赖里有 profileinstaller，所以应该已预编译”更可靠。包内没有配置文件不等于设备完全没有 ART 优化：后台编译、运行积累的配置仍可能存在，但当前没有目标设备证据。不能直接将所有顿挫都归因于 JIT。

建议先准备同安装身份、可覆盖升级的非 debuggable release 性能候选，启用 R8 并检查反射、资源读取、更新安装等功能，再加入覆盖冷启动和首页滑动的应用 Baseline Profile，检查实际 APK 条目和设备安装状态。Android 官方也建议先核对 release/R8 与 Profile 配置，再评价 Compose 性能。[官方说明](https://developer.android.com/develop/ui/compose/performance)

验收：同一设备、相同商品/账号/外观参数，分别记录当前发布包与优化候选的冷启动及前几次滑动，分开呈现编译状态，不以 APK 大小或单次 `am start -W` 代替帧表现。

### 2. P1：首段滑动触发了内容组合、列表测量和文字排版

调用链：

```text
手指向上滑
  → PageChromeState.onPreScroll 修改 offset（0 → 128dp）
  → AnimatedContent 内容读取 offset 计算 topInset
  → ShopPage 接收变化的 topInset
  → LazyColumn 的 contentPadding 每帧改变
  → 同时重算商品筛选/分类/分组及可见布局

offset 超过 68dp
  → 标题字号、行高、字距及容器尺寸也逐帧改变
```

位置：

- [CollapsingChrome.kt](../../android-app/ui-lab/src/main/java/com/deuterium/app/uilab/CollapsingChrome.kt) 第 29–40 行：滚动直接写入折叠状态；第 51–69 行：组合阶段读取状态，调整高度、padding、字号、行高、字距，并在变化尺寸的 `BoxWithConstraints` 内布局标题。
- [DeuteriumActivity.kt](../../android-app/ui-lab/src/main/java/com/deuterium/app/uilab/DeuteriumActivity.kt) 第 235–242 行：当前页面内容在组合阶段读取 `chrome.offset` 并传入变化的 `inset`。
- [Storefront.kt](../../android-app/ui-lab/src/main/java/com/deuterium/app/uilab/Storefront.kt) 第 34–54 行：`ShopPage` 每次执行重新筛选商品、获取分类、拆分双列数据，同时把变化的顶部间距传给 `LazyColumn`。

折叠达到 128dp 后，`offset` 不再随继续上滑变化，这条额外路径停止更新。因此，“刚开始滑比较顿，标题收起来后好一些”与此机制相符。这是根据代码的推断，尚非真机因果结论。也不意味着所有不可见商品都被绘制：现有列表仍然是 lazy 布局。

在严格同效约束下，先将商品筛选、分类与行分组的依赖限制为商品快照/筛选条件，隔离折叠状态引发的外围重组。**保留标题每个进度对应的字号、行高、字距及位置公式，不采用双标题缩放/交叉淡化替代原排版。** 部分文字测量本身是现有效果所需工作，不承诺能无损删除。只有能保持测量结果、像素取整、滚动范围、触摸坐标和安全区域的阶段迁移才可进入实现，不能直接把动态 padding 替换为视觉位移。[Compose 阶段](https://developer.android.com/develop/ui/compose/phases?hl=en)、[推迟状态读取](https://developer.android.com/develop/ui/compose/performance/bestpractices?hl=en)

验收：分别测“从完全展开到折叠”和“已折叠后的持续滑动”；商品数据不变时，折叠过程不再重跑商品筛选和分类。检查标题端点、首屏布局、滑回顶部、快速反向滑动和大字号。

### 3. P2：徽标还没显示，收缩动画就可能已经开始

- [DeuteriumActivity.kt](../../android-app/ui-lab/src/main/java/com/deuterium/app/uilab/DeuteriumActivity.kt) 第 65–77 行：获得焦点或系统 splash 退出回调后立即设置 `nativeLaunchReady`；退出回调直接移除系统 splash。
- [AppLaunchOverlay.kt](../../android-app/ui-lab/src/main/java/com/deuterium/app/uilab/AppLaunchOverlay.kt) 第 33–43 行：仅根据 `nativeReady` 启动 620ms 位移/缩放，根据 `ready` 和计时开始 320ms 淡出。
- [AppIdentity.kt](../../android-app/ui-lab/src/main/java/com/deuterium/app/uilab/AppIdentity.kt) 第 30–45 行：徽标通过异步图片请求显示，没有将 `onSuccess` 或首个可绘制帧反馈给开屏时序，也没有同步衔接用的占位徽标。

如果图片解码/调度迟到，第一张可见徽标已经处在收退动画的中间位置；即使没有长帧，用户也可能感到“跳了一下”。资源包存在时，`source` 还会从内置资源切换到文件，可能发生第二次请求或图像替换。是否命中缓存会改变实际表现。

优先检查同一选定素材能否提前预热并在系统层/应用层交接时复用，保留原始位置、缩放曲线及 620ms/320ms 时长。**新增等待、改变启动条件或替换占位图不能直接作为等效优化**，必须先证明不会改变用户可见的启动时序。保留资源包回退和暖启动/覆盖安装不依赖系统回调的修复，也不能恢复主线程大图同步解码。

验收：缓存命中、进程冷启动、图片加载延迟、资源包可用/缺失/损坏、暖启动和关闭动效。逐帧检查首个应用徽标的中心、大小与后续位移是否连续，同时采集 FrameTimeline，区分“视觉跳变”和“真的漏帧”。

### 4. P2：进入首页时会刷新两遍商城

[Storefront.kt](../../android-app/ui-lab/src/main/java/com/deuterium/app/uilab/Storefront.kt) 第 36 行的 `LaunchedEffect(Unit)` 调用 `refreshStore()`；[LabState.kt](../../android-app/ui-lab/src/main/java/com/deuterium/app/uilab/LabState.kt) 第 113 行在会话验证后再次调用。

[BackendCatalog.kt](../../android-app/ui-lab/src/main/java/com/deuterium/app/uilab/BackendCatalog.kt) 第 60–66 行没有共享在途请求/刷新去重。每次依次获取分类、品牌、商品，再刷新购物袋。在四个接口都只有一页且成功的正常路径上，两条入口合计执行 **8 次请求，而单次完整刷新只需 4 次**；分页会更多。这是调用路径推导，没有声称已经抓取了手机网络请求。

两次结果还会分别 `ShopCatalog.clear()` / `addAll()` 并重新应用购物袋。Compose 可能合并同一批状态通知，因此不能据此断言一定出现中间空白帧，但重复网络、数据映射及状态更新明确存在。接口响应落在首屏动画或第一次滑动附近时会增加竞争。

优先研究同账号、同参数、实际重叠的读取是否可共享同一次在途结果，避免只用互斥锁把两遍请求变成串行执行。保持页面进入、用户显式刷新及商品版本变化后的刷新语义，不引入 TTL 缓存或减少独立刷新次数来牺牲及时性；成功、失败及后返回结果的处理顺序须验证一致。

验收：使用请求计数确认首次进入每条路径只刷新一轮；再检查并发刷新、失败重试、切回商城和商品修改后的刷新。

### 5. P2：启动就绪不代表首页就绪，数据加工仍集中在主线程

[LabSession.kt](../../android-app/ui-lab/src/main/java/com/deuterium/app/uilab/LabSession.kt) 第 20–28 行使用 `Dispatchers.Main.immediate`，先设置 `restoring=false`、`loadedUser=name`，才执行 `connect()`。因此 `session.readyFor(userName)` 表示本地状态对象已建立，不代表商城数据或首屏图片已准备好。

与此同时，[DeuteriumActivity.kt](../../android-app/ui-lab/src/main/java/com/deuterium/app/uilab/DeuteriumActivity.kt) 第 115–119、167 行让启动遮罩等待付款声音准备和 `FacePayAssets.load()`；[FacePayAssets.kt](../../android-app/ui-lab/src/main/java/com/deuterium/app/uilab/FacePayAssets.kt) 第 25–57 行会解析并解码支付图集。该图集源尺寸 1512×1260，按 RGBA 估算约 **7.27 MiB**，此数是源尺寸估算，非实测堆分配。解码已经在后台，问题是它占据启动关键路径，却不属于当前首页必需内容。

[LabState.kt](../../android-app/ui-lab/src/main/java/com/deuterium/app/uilab/LabState.kt) 第 102–119 行在会话验证后立即启动钱包/账单、聊天历史/目录、商城、市场、委托、订单、通知等工作。网络和顶层 JSON 解析已在 [BackendApi.kt](../../android-app/ui-lab/src/main/java/com/deuterium/app/uilab/BackendApi.kt) 第 84–112 行的 IO 调度器执行，**这里不存在“所有网络都阻塞主线程”的证据**。但返回调用方后的对象映射、逐项查找/排序和状态提交继续使用主线程，例如：

- `LabState.kt` 第 140–162 行：首次读取最多 100 条公共历史，逐条 `addMessage`，每条都查重、转换时间、追加并 `sortBy`。
- `BackendCatalog.kt` 第 146 行：全量订单拉取完成后清空列表，逐条 `putOrder`。
- [BackendCommissions.kt](../../android-app/ui-lab/src/main/java/com/deuterium/app/uilab/BackendCommissions.kt) 第 28–35 行：逐条插入并每次重新排序整个委托列表。

当前数据量下各段是否超过一帧尚未测量；但随着历史增多，逐条更新和反复排序会放大首屏附近的工作量。初始内容还可能在遮罩淡出后才到达，触发布局与纹理上传。

严格同效方案首先保留现有启动就绪条件、预热和刷新触发点，优化批量历史转换、去重、排序与状态提交的内部实现，并验证批次期间的可见状态及列表动效。**移除支付素材就绪条件、推迟非当前页面请求或新增首屏占位，暂不列为直接实施方案**，因为可能改变支付首次进入、红点、页面切换及加载时序。未知资金结果恢复、鉴权和必要通知保持原有行为。

验收：记录“遮罩退场”“首屏内容稳定”“首个可响应滑动”的不同时间；用固定数据和相同网络条件测试批量历史回填，确认资金恢复仍按原请求身份工作。

## 待测热点与已经排除的误判

- **玻璃 GPU 成本待测。** [LiquidGlass.kt](../../android-app/ui-lab/src/main/java/com/deuterium/app/uilab/LiquidGlass.kt) 第 57–87 行创建/使用 RuntimeShader、RenderEffect 和背景采样；[GradientGlassHeader.kt](../../android-app/ui-lab/src/main/java/com/deuterium/app/uilab/GradientGlassHeader.kt) 第 25–43 行仍有模糊与离屏遮罩。首次 shader 使用、图片纹理上传以及滚动时的采样可能增加渲染成本，但没有 RenderThread/GPU 轨迹，不能将玻璃定为唯一原因，更不能把默认关闭玻璃当成验收通过。
- **上一轮已修正的开屏每帧组合读取没有恢复。** `AppLaunchOverlay.kt` 第 50–59 行仍在 `graphicsLayer` 读取位移、缩放和透明度；加载线的状态在 Canvas 绘制阶段读取。
- **商品图有异步及缓存基础。** `CachedPhoto.kt` 使用 `AsyncImage`；`AppImages.kt` 复用内存/磁盘缓存；`ImageCacheStore.kt` 有 IO 切换、同键互斥及 3 个并发下载限制。未发现首页同步完整下载/解码商品图的实现，纹理上传与缓存命中仍需实测。
- **顶部模糊已缩小采样高度。** 现代码记录顶部区域加模糊邻域，不能把旧版整屏模糊问题重复写成当前新发现。
- **首帧前仍有同步初始化候选。** `DeuteriumActivity.onCreate` 创建 `BackendApi` 时会恢复 Keystore 会话；`AppUpdates.initialize` 可能同步读取资源包文案。它们更直接影响系统开屏停留时间，不能未经追踪就解释为收退阶段卡顿。
- 本次本地历史检索未提供超出发布文档的有效性能测量证据。没有沿用历史“启动成功”结果推断流畅。

## 下一轮测量与验收

| 场景 | 控制条件 | 要观察的证据 |
| --- | --- | --- |
| 冷启动与徽标收退 | 同机、同刷新率、固定温度区间、相同账号和素材；分别记录图片缓存/ART 编译状态 | 系统 splash 交接、徽标首个可见帧、主线程长切片、FrameTimeline、JIT、GC、纹理上传 |
| 首次向上滑动 | 从顶部开始，单独标记前 128dp 折叠区间；与已折叠后滑动分开 | 商品筛选执行次数、重组/测量耗时、标题排版、长帧连续长度 |
| 进入后 10 秒 | 首屏数据与聊天历史数量固定 | 重复请求计数、批量回填时间、是否与首滑重叠 |
| 对照诊断 | 当前外观为正式验收基准；玻璃/动效开关只作为对照变量 | 开关前后 CPU/GPU 差异，不用关效果替代原设计验收 |
| 改动后回归 | 浅/深色、大字号、返回顶部、导航切换、覆盖安装 | 视觉和触摸不回退、启动不会被无网络/坏图片永久拦住 |

每种场景重复采样，报告帧耗时分位数、超预算帧占比、连续长帧及对应轨迹。60Hz/120Hz 名义周期分别约 16.67ms/8.33ms，实际判定以设备 FrameTimeline 的时限为准。`am start -W` 可记录启动冒烟，但不能代替徽标和首页帧轨迹；首帧显示与完整内容可用也应分别记录。[Android 启动测量说明](https://developer.android.com/topic/performance/vitals/launch-time)

下一步按全 App 复核中的严格同效边界建立性能候选，先处理可保持原输出的内部开销；任何涉及渲染或时序的改变先做等效对照。是否消除用户感知的顿挫，要以目标手机相同场景的对照结果确认。
