# App 2.0.10 全范围性能复核：严格保持原效果

> 后续实施与实际证据见 [性能优化验收](../qa/app-performance-equivalence-v210.md)。下文保留审查阶段事实与候选边界，未采用的候选不计为已完成优化。

2026-09-11。用户追加要求：**其他地方的性能也要 review，前提是严格保证效果实现完全一样。** 本文继承 [启动与首页审查](app-performance-v210-2026-09-11.md)，并收紧其中可能改变效果的建议。

审查范围为当前发布模块 `android-app/ui-lab`，包含四个一级页面及其聊天、AI、订单、钱包、转账、委托、退款、图片、通知、设置、更新与公共渲染链路。对 91 个主模块 Kotlin 文件做性能模式扫描，并沿主要页面调用链复核；保留的旧 `android-app/app` 不作为当前发布性能对象。

本轮仍为源码与产物审查，没有修改 App 代码或资源。新增了扩展报告和 [效果基线清单](artifacts/app-performance-v210/effect-baseline.json)，清单记录源码/资源摘要及动效位置，用于后续追溯，**不是已通过逐帧等效测试的证明**。ADB 检查仍无设备，实际帧耗时、耗电和内存峰值尚未测量。

## 必须保持的效果与行为

这里的优化目标是减少重复工作、主线程阻塞和漏帧。同一输入、配置与动画进度对应的画面和行为必须保持一致，不能把“看起来差不多”当作通过。

| 保持项 | 具体边界 |
| --- | --- |
| 静态画面 | 布局、尺寸、间距、文字、字体、字号、字重、行高、字距、换行、颜色、渐变、圆角、阴影、裁切、图片内容和采样质量 |
| 动态画面 | 原轨迹、时长、延迟、缓动、弹簧参数、缩放、透明度、首尾衔接；不能用少帧、降刷新频率或另一种近似动画替代 |
| 玻璃 | 同样的背景内容、采样坐标、模糊、折射、高光、渐隐、倾斜响应；所有可调参数和开关继续有效 |
| 交互 | 点击与拖动命中区域、键盘、焦点、返回、滚动位置、滚动范围、列表锚点、选择/复制/长按、无障碍语义 |
| 业务与异步状态 | 相同请求身份、落盘顺序、重复提交防护、结果未知恢复、账号隔离、消息顺序、未读/通知、刷新及时性、错误和重试表现 |
| 付款体验 | 1–3 秒模拟识别及与提交并行的现有关系；服务端成功后才显示成功；原声音、振动、完成回调和自动关闭节点 |

源代码里仅有未调用的旧组件，不作为“当前一定卡”的证据。例如 `AppLauncherIcon` 和 `ProductPhoto` 在主模块中未找到调用点；关于页实际调用异步 `LoginVersionBadge`，不能把旧的同步 App 图标函数列为当前关于页必经热点。

## 已撤回或暂不采用的方案

- **撤回双标题缩放/交叉淡化替代字号变化。** 缩放后的字形、hinting、字距和中间帧可能不同，不能保证原效果。
- 不降低图片分辨率、编码质量、`FilterQuality.High`，不替换原图；不降低玻璃模糊/折射，也不进一步裁掉未经像素验证的采样邻域。
- 不改变动画时间、弹簧参数、付款声音/振动节点；不把动态背景冻结为截图，不降低传感器频率，不跳过支付素材帧。
- 不增加 AI 分段缓冲来改变流式文字显示节奏；不把 Markdown 暂时改成纯文本或简化渲染。
- 不直接推迟非当前页请求、放宽轮询间隔、移除支付素材就绪条件或引入更旧的缓存结果。它们可能影响进入页面、红点和加载状态。
- 不将资金/消息请求身份的 `commit()` 批量换成 `apply()`，不通过少做校验、取消恢复或清除记录换速度。
- 不给可变业务对象盲目标注 `@Stable`/`@Immutable` 以强行跳过重组，也不借优化升级 Compose/Coil/Markwon 来改变渲染基线。

## 新发现与处理顺序

P1 表示优先处理；P2 是明确额外工作或需要限定场景的热点；P3 为较小的无谓分配。以下均为**代码事实及优化候选**，实际收益未测。候选也必须经过等效验证，不能仅凭分类批准发布。

| ID | 优先级 | 范围 | 发现 |
| --- | --- | --- | --- |
| F1 | P1 | 信息/联系人 | 所有联系人塞入同一个 lazy item，进入可见区时一次性创建整组内容 |
| F2 | P1 | 聊天/付款/账号 | 请求身份同步落盘、登录会话加密存在主线程调用路径 |
| F3 | P2 | 全部远程图片 | 图片授权表扫描整段 JSON 时持锁，界面读图片信息争用同一把锁 |
| F4 | P2 | 加购/导航/开关 | 部分高频动画值在过大的组合范围读取，引发外围重组 |
| F5 | P2 | AI 流式回复 | 每段回复重复构造 JSON、转换固定时间/身份/来源，再转换回消息模型 |
| F6 | P2 | 消息/订单/委托 | 批量历史逐项查重、插入和重新排序；反复刷新也重复加工未变化内容 |
| F7 | P2 | 首页/市场/信息/账单 | 无关状态变化触发筛选、排序、分组和联系人历史遍历 |
| F8 | P2 | 页面后台运行 | 多个页面轮询不受生命周期约束；联系人还有两套刷新入口 |
| F9 | P2 | 图片上传 | 将最大 20MB 图片完整读成字节数组并复制，上传期间保留大缓冲 |
| F10 | P3 | 玻璃/头像裁切 | 绘制时重复创建与当前动画进度无关的渐变和路径 |
| F11 | P3 | 通知设置 | 一次响应逐个设置约 10 个偏好，每个键分别安排持久化 |

### F1：联系人列表没有真正按可见行创建

[InformationPages.kt](../../android-app/ui-lab/src/main/java/com/deuterium/app/uilab/InformationPages.kt) 第 51–69 行：`LazyColumn` 的单个 `item` 中放入 `Surface → Column → people.forEachIndexed`。内部 `key` 只保留组合身份，不能让这些行自动获得 lazy 能力。因此联系人越多，进入该卡片时一次性组合、测量和头像请求的数量越多；总数受用户联系人数据影响。

同效候选：将联系人改为按行 lazy 创建，同时保留整组连贯卡片外观、首尾圆角、分隔线、零额外行距和原行高度。**不能简单换成一堆独立圆角卡片。** 优先先缓存联系人计算；行级虚拟化因涉及测量和滚动锚点，必须逐像素、触摸和滚动验证后才能接受。

对照场景：0/1/多联系人、长名字、特别关心置顶、上下边缘被裁切、快速滑动、资料头像点击、滚动中收到新消息后排序变化。检查头像首次出现状态，不能留下新的空白闪烁。

### F2：同步落盘可能卡住点击后的动画与输入

[BackendApi.kt](../../android-app/ui-lab/src/main/java/com/deuterium/app/uilab/BackendApi.kt) 第 118–155 行，`saveTransfer`、`savePendingChat/Direct/Forward/AI`、`saveOperation`、`saveUploadState` 的非空写入使用同步 `commit()`。例如 [LabState.kt](../../android-app/ui-lab/src/main/java/com/deuterium/app/uilab/LabState.kt) 第 242–245、396–400 行，以及 [BackendAI.kt](../../android-app/ui-lab/src/main/java/com/deuterium/app/uilab/BackendAI.kt) 第 165、177 行，存在从主协程直接调用的路径。登录/注册返回后还会在 `BackendApi.kt` 第 56–70 行进行 Keystore 加密。不是所有调用都在主线程：商品图片编辑的上传外层已经是 IO，需逐调用点评估。

同步写会占用调用线程，并可能等待同一偏好文件上尚未完成的异步写；这是 Android 官方明确提示的 UI 阻塞来源。[SharedPreferences 说明](https://developer.android.com/training/data-storage/shared-preferences?authuser=1)、[Editor API](https://developer.android.com/reference/android/content/SharedPreferences.Editor.html?authuser=6)

同效候选：把真正的磁盘/加密步骤迁到后台并等待完成，保持“原请求身份写入完成 → 发出请求”的顺序。**新增挂起点之前必须建立原有防重入边界**，否则连续点击可能在 `busy` 尚未设置时进入第二次请求；还要保持账号切换、失败、取消、重建后的恢复语义。不能只替换存储 API 就声称完成。

对照场景：慢盘、并发设置写入、连续点击、请求前后杀进程、重进页面、账号切换、结果未知；相同随机识别时长和服务响应条件下比较支付画面、声音与振动节点。

### F3：后台解析可能通过锁间接挡住界面

[RemoteImageUrls.kt](../../android-app/ui-lab/src/main/java/com/deuterium/app/uilab/RemoteImageUrls.kt) 第 14–32 行，`remember` 使用 `@Synchronized`，并在方法内递归遍历整段 JSON、解析 URL/时间、查询和更新授权项。[BackendApi.kt](../../android-app/ui-lab/src/main/java/com/deuterium/app/uilab/BackendApi.kt) 第 107 行在 IO 请求中调用它；但 `key/access/refreshPath` 第 34–39 行也同步在同一对象上。[CachedPhoto.kt](../../android-app/ui-lab/src/main/java/com/deuterium/app/uilab/CachedPhoto.kt) 第 22–23 行在组合阶段调用这类读取。

所以“解析在 IO”仍不能保证 UI 不等待它。大订单/消息响应处理期间，图片读取存在等锁路径；没有真机 monitor contention 数据，不能报告已经测得阻塞多少毫秒。

同效候选：把纯 JSON 遍历和可独立完成的转换移出共享锁，锁内只做必要的原子检查/提交；保持原先同响应内覆盖次序、过期状态、保留期限、账号隔离和退出清理语义。不能随意缓存授权读取结果或删除同步，避免旧授权重新覆盖已失效记录。

### F4：加购动画读取位置过高，导航和开关也有额外重组

- [DeuteriumActivity.kt](../../android-app/ui-lab/src/main/java/com/deuterium/app/uilab/DeuteriumActivity.kt) 第 301 行在 `LabApp` 读取 `flightProgress.value`，再传给 [ProductMedia.kt](../../android-app/ui-lab/src/main/java/com/deuterium/app/uilab/ProductMedia.kt) 第 39–46 行。图层内部虽已处理位移/缩放，父级仍逐帧订阅了进度。
- [MovingGlassNav.kt](../../android-app/ui-lab/src/main/java/com/deuterium/app/uilab/MovingGlassNav.kt) 第 30–32 行为计算拉伸，在组合阶段读取位置动画。不能据此断言所有导航子项都重新绘制，但作用范围超过移动指示器。
- [IosControls.kt](../../android-app/ui-lab/src/main/java/com/deuterium/app/uilab/IosControls.kt) 第 45–49 行，开关用动画值生成背景和 `offset(Dp)`，每帧进入组合/布局。

同效候选：加购传递进度提供器，在原 `graphicsLayer` 内读取，原弧线公式、浮点位移、72dp 图像、缩放/透明度公式和 650ms 缓动完全保留；导航把原位置/拉伸计算限制到指示器组合范围，不合并或改写两套弹簧。开关保留原布局位移的像素取整，不能直接换成亚像素图层位移后当作相同。

对照场景：起止点、中途点击/拖动/反向、加购时收到通知、快速切导航、开关连续点按、动效关闭。只减少外围失效，不改变已有动画。阶段优化的依据见 [Compose 阶段与性能](https://developer.android.com/develop/ui/compose/performance/phases)。

### F5：AI 每段回复都在加工固定元数据

[BackendAI.kt](../../android-app/ui-lab/src/main/java/com/deuterium/app/uilab/BackendAI.kt) 第 167–179 行，每个 `delta` 都构造完整消息 JSON 和来源 JSON，再调用第 124–129 行 `put`：重新解析时间、生成稳定 UUID、格式化时间、查找消息位置并构造 `ChatLine`。第 203 行逐事件切换到主线程执行这些工作。这里的 `runBlocking` 位于 OkHttp 回调线程，**不能误报成在主线程 `runBlocking` 等待网络**。

同效候选：同一助手消息固定的 ID/创建时间/显示时间只转换一次；每个原始事件仍按原顺序即时更新同一条消息的正文、状态和来源，保持 `/new`、中断、恢复和 `done` 权威消息覆盖逻辑。不要增设 50/100ms 缓冲来减少更新次数。

[MarkdownBody.kt](../../android-app/ui-lab/src/main/java/com/deuterium/app/uilab/MarkdownBody.kt) 第 44–77 行已经缓存 renderer 和解析后的文本，`view.tag` 也防止同结果重复 `setParsedMarkdown`；因此“所有历史消息每次都重解析”不成立。正在增长的回复确实会重新解析全文，但在严格同效下不能直接替换解析器或改成纯文本；长表格/代码块的实际耗时列为待测。

### F6：批量数据路径反复排序，重复刷新反复加工

`LabState.kt` 第 140–162 行及 374–375、385 行的聊天历史，`BackendCatalog.kt` 第 130–146 行的订单，`BackendCommissions.kt` 第 28–35 行的委托，均有逐项查找/插入和排序路径。联系人每轮在 `LabState.kt` 第 329–336、351–353 行执行移除/添加玩家。订单 5 秒详情刷新也会重新排序整个订单集合。

同效候选：在后台转换不可变快照，采用同一规则批量去重/排序并提交；未变化的值保持原对象/状态。保留所有复合排序键、时间并列时的稳定顺序、重复 ID 的覆盖选择、隐藏记录、序号、聊天插入动画、未读数量和自动滚动。**不得仅比较最终数量**；异常发生在批次中途时，也须保持同一可见结果与失败反馈，不能悄悄改成全批失败或另一种部分更新。

这项与首次启动相关，但同样影响翻历史、打开订单列表、刷新委托及聊天重连。不能通过截断用户已加载历史或减少数据范围来优化。

### F7：多个页面的派生列表缺少按输入复用

- `Storefront.kt` 第 37–48 行、`MarketplacePages.kt` 第 35 行：顶部折叠等无关变化时重新筛选/分组。
- `InformationPages.kt` 第 51 行调用 [RecentContacts.kt](../../android-app/ui-lab/src/main/java/com/deuterium/app/uilab/RecentContacts.kt) 第 6–11 行：逐联系人过滤并遍历已加载私聊历史，再排序联系人；信息页也使用相同折叠头部。
- [BillHistoryPage.kt](../../android-app/ui-lab/src/main/java/com/deuterium/app/uilab/BillHistoryPage.kt) 第 50–69 行：过滤、排序、汇总、日期分组随页面执行重算。分页是 25 条，已存在，不应误报为一次拉全库。
- `OrdersPage`、`CommissionHallPage` 同样在组合中排序；列表已有业务 key，主要问题并非“所有列表都没 key”。

同效候选：按真正的数据、筛选、账号、时区/日期等依赖缓存结果；对 `SnapshotStateList` 不能只 `remember(原列表对象)`，否则内容变了对象没变会显示旧结果。聊天翻历史触发中读取 `firstVisibleItemIndex` 也可隔离到小范围观察组件，但保留原阈值和触发语义。

验收：缓存前后完整有序结果一致，包含金额符号、零额/大额、同时间记录、时区/跨日、历史翻页、新消息置顶和筛选取消。钱包大额排版使用 `rememberTextMeasurer`，本身有缓存基础；不能为了省测量删除大额自适应或修改换行规则。

### F8：页面轮询在切后台后仍可能运行

`InformationPages.kt` 第 34 行有 15 秒联系人刷新，第 99 行有 4 秒私聊/AI 刷新；`OrderPages.kt` 第 62 行、`CommissionPages.kt` 第 94 行、`RefundPages.kt` 第 52 行有 5 秒详情刷新。它们由 `LaunchedEffect` 生命周期控制，没有绑定 Activity 的 RESUMED/STARTED 状态，Activity 进入后台而组合未销毁时可能继续请求。私聊已读上报另有前台判断，不能把这一点混为后台自动已读。

另有 `LabState.kt` 第 111 行的前台 4 秒联系人循环；联系人锁目前把并发刷新排队执行，没有共享一次结果。这是后台网络/电量与回前台数据加工的候选热点，不代表已经测得耗电。

**本项暂不直接改刷新策略。** 先记录现有请求与前后台行为；仅在能保持原前台更新时刻、通知及时性、恢复入口和回前台内容的前提下收敛重复工作。不能统一暂停全部协程、WebSocket、AI 正在进行的回复、下载或未知交易恢复。不能为减少网络请求而放宽现有轮询间隔。

### F9：图片上传存在可降低的字节缓冲峰值

[BackendAssets.kt](../../android-app/ui-lab/src/main/java/com/deuterium/app/uilab/BackendAssets.kt) 第 28–42 行用 `ByteArrayOutputStream` 读取最多 20MB，然后 `toByteArray()` 复制，`data` 又随上传、验证和缓存写入持有至调用结束；第 74 行用完整数组创建请求体。多图编辑是顺序上传，并非 5/10 张一起解码，不应夸大成同时占用数百 MB。头像裁切、商品拷贝及真正上传请求已在后台执行。

同效候选：利用现有私有临时文件流式计算摘要/上传/写缓存，减少重复的大字节数组。维持相同原始字节、MD5/SHA-256、MIME、大小、授权、上传 ID 和验证重试；不重新压缩/缩小图片。需要验证清理缓存、取消或重试时临时文件仍受正确保护，不能把内存优化变成上传中途丢文件。

512×512 的小祥头像仍有同步 `painterResource` 首次解码路径，源文件 296,037 字节、RGBA 估算 1MiB；它是 Saki 首次进入的待测项。严格同效下不能直接改成首次空白的异步图片，也不能擅自降采样。主模块没有发现“一进入商品页就同步解码所有服务器原图”的实现。

### F10：静态绘图对象可复用，动态材质参数必须原样保留

[LiquidGlass.kt](../../android-app/ui-lab/src/main/java/com/deuterium/app/uilab/LiquidGlass.kt) 第 129–136 行在绘制中重新构造三组尺寸/主题决定的背景渐变；第 64–69 行创建的 `rim` 没有被使用。[AvatarCropper.kt](../../android-app/ui-lab/src/main/java/com/deuterium/app/uilab/AvatarCropper.kt) 第 83–86 行在手势重绘时重建固定遮罩路径和 ImageBitmap 包装；`asImageBitmap()` 是包装复用，不应误写为每帧复制整张位图。

同效候选：缓存与尺寸、密度、主题、资源内容相关的原始 Brush/Path/包装对象，删除未使用的对象构造。保持原 shader、绘制次序、混合方式、裁切和参数。倾斜相关渐变仍随原传感器值变化，不能缓存成静态；也不要把高频状态放进缓存构造导致整组 blur/shader 反复重建。[Android 绘制缓存说明](https://developer.android.com/develop/ui/compose/graphics/draw/modifiers)

### F11：通知偏好一次更新触发多次写入安排

[NotificationPreferences.kt](../../android-app/ui-lab/src/main/java/com/deuterium/app/uilab/NotificationPreferences.kt) 第 24–36 行：8 种通知加总开关/预览共约 10 个键，`apply(value)` 逐个调用 `set`，每个 `set` 都创建 editor 并 `apply()`；一次修改还可能先后应用读取结果和更新结果。Android 可以合并部分磁盘工作，因此不能声称每次必定发生 10 次物理 fsync。

同效候选：同一服务响应使用一个 editor 写入实际变化的键，保持同一批 Compose 状态、原值回滚/失败处理、后台通知读取到的完整配置与账号隔离。避免改成延迟持久化或改变开关动画；普通通知偏好与请求身份的耐久化不能混用同一种简化。

## 各范围的审查结论

| 范围 | 已核对 | 结论/限制 |
| --- | --- | --- |
| 启动/登录/安装返回 | 原生 splash、徽标、会话、支付素材预热、别名恢复 | 继承启动报告，严格保留暖启动/覆盖安装修复；F2 |
| 商城/市场/搜索/加购 | 卡片、筛选、列表、图库、购物袋飞行动画 | F4/F7；原商品图、购物袋轨迹和报价触发保留 |
| 信息/公共/私聊 | 联系人、分页、输入、引用、发送、自动滚动、红点 | F1/F2/F6/F7/F8；现有 lazy message key 和输入点动画保留 |
| Saki/Markdown | 流式事件、解析缓存、来源、套餐与购买确认 | F5；不得缓冲改变显示节奏；Saki 弹层时间参数保留 |
| 钱包/账单/转账 | 刷新、排序、分页、日期、大额排版、提交 | F2/F7；余额刷新角度已经在图层读取，不重复修 |
| 订单/委托/退款/介入 | 列表、详情刷新、倒计时、操作反馈与图片凭证 | F6/F8；计时读取主要位于对应 lazy item，不能误报整页每秒重组 |
| 图片/头像/上传/缓存 | 解码线程、字节缓冲、缩放、裁切、授权锁 | F3/F9/F10；保留原像素、裁切数学和账号/过期隔离 |
| 玻璃/导航/公共控件 | shader、采样、传感器、开关、分段、按钮、浮层 | F4/F10；GPU 时间待测，不默认关效果；MotionButton/SegmentedControl 已主要使用图层动画 |
| 通知/优惠券 | 应用内卡片、去重、红点、系统偏好 | F11；优惠卡和 ticker 已跟随前台生命周期，不能扩大为“全部轮询未暂停” |
| 存储/更新 | 目录计量、清理、下载进度、校验与安装 | 计量/校验/拷贝已在 IO，下载进度约 80ms 更新；不降低频率、不跳完整性/签名检查 |
| 内存/生命周期 | 缓存上限、ViewModel 清理、传感器、Context | 图片缓存有预算，AppUpdates/LabSession 有取消，传感器暂停会注销；F8/F9 之外的实际泄漏需要堆/运行数据，未作“无泄漏”保证 |

## 原效果基线与后续验收门槛

基线使用发布源码 `65c08a290896…` 对应的当前效果；审查 checkout `ad8a8ef` 的 Android 内容与它相同。原 APK 摘要和 debug/Profile 事实见 [产物证据](artifacts/app-performance-v210/apk-audit.json)。新增 [效果基线清单](artifacts/app-performance-v210/effect-baseline.json) 包含主模块源码/资源哈希及关键效果原文片段，防止后续把改动后的效果误当成原基线。

部分必须保持的原参数：

| 效果 | 当前值或规则 |
| --- | --- |
| 开屏 | 620ms 收退、320ms 淡出；104dp 徽标、192/104 初始缩放、向上 94dp；原缓动与进入条件 |
| 页面切换 | 淡入 180ms/延迟 25ms，位移 260ms/宽度 1/14，淡出 130ms |
| 加购 | 650ms FastOutSlowIn；原正弦弧线及缩放/透明度公式 |
| 导航 | 位置 spring(.72,380)，拉伸 spring(.70,450)，原拖动公式 |
| 开关/按钮 | 开关 spring(.8,600)，按钮 spring(.8,650)、按下缩放 .97；原位移取整 |
| 通用浮窗 | 180ms 进入、130ms 退出、dim .18；原玻璃和窗口行为 |
| 付款 | 1–3 秒模拟识别、原 frame 进度/素材/High 采样、140/130ms 完成阶段延时和 850ms 自动关闭；关闭动效分支另按原实现 |
| 默认玻璃 | 底栏 4/.12/24/.30，浮层 36/.65/24/.30；头部 blur18/fade32；用户已保存参数优先 |

验收分为三个部分，缺任一部分不得称“严格同效优化已通过”：

1. **画面与动效对照。** 同设备、系统、字体、密度、主题、真实材质参数、输入数据与动画进度下，静态逐像素比较，动态覆盖起点/中间/终点及打断/反向过程。任何稳定可复现的画面差异都视为不等效；先用基线自身重复运行区分环境噪声，不用宽松相似度阈值掩盖差异。
2. **行为与事件对照。** 比较命中区域、滚动位置/锚点、键盘与返回、消息顺序/未读、后台通知、错误与重试。固定测试随机识别时长和服务响应后，声音、振动、成功勾、提交/落盘/恢复的事件偏序和显式延时与原实现相同。
3. **性能对照。** 当前包与同源码性能候选在相同编译/缓存条件下采集 FrameTimeline、主线程/RenderThread、锁等待、I/O、JIT/GC、内存峰值；区分冷/暖启动、首滑/持续滑动。只报告对应设备和场景的实际收益，不用测试通过、源码更短或 APK 更小替代结果。

有限场景测试不能证明所有设备上的逐像素一致，因此报告应明确验收设备、覆盖场景和未测范围，不能提前给出“任何设备上 100% 一样”的结论。本轮 App 源码/资源未变，保持原实现；上述优化均尚未实施。

建议第一批选择 F4 中的加购状态读取下沉、F10 中的静态对象复用、F11 的同批偏好写入和 F7 的完整依赖缓存；它们较容易建立等效证明。F1/F2/F3/F5/F6/F8/F9 涉及布局、并发或状态时序，分别做有针对性的对照。release/R8/Profile 性能候选同样必须保留原素材/库版本，并通过完整行为与画面对照后才可交付。
