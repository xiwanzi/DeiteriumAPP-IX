# App 性能优化与同效验收

2026-09-11。用户授权实施，硬约束为相同画面、动效和交互。维护工作区 `C:/DeiteriumAPP-IX/.worktrees/performance-audit-v210`，分支 `xiwanzi/performance-equivalence-v210`，基于 `ad8a8ef`（Android 源码与已发布 `65c08a290896` 相同）。本轮为本机候选，版本仍为 2.0.10（21000），未推送、合并或部署。

## 最终候选

[APK](../../delivery/Deuterium-2.0.10-performance/Deuterium-2.0.10-21000-performance.apk) 为 **8,875,538 字节**（8.88 MB），SHA-256 `78d5b52f3d821d516361fb4dce9f532c5bf8e5b083309c95bf91553d04dacdaf`。相较原发布 APK 的 23,557,745 字节减少 **62.3%**；这是安装包体积变化，不是启动或帧率提升百分比。DEX 由 65,452,544 字节降至 16,477,664 字节。

版本号/包名与原版一致；证书 SHA-256 为 `04a213066a78a968c2a1328a153a0967f99d117e0a826285e3f4af705db6f2f3`。`apksigner` 验证通过，设备覆盖安装通过，应用 flags 不含 DEBUGGABLE。未建立线上更新条目；本地 `delivery/Deuterium-2.0.10-performance/` 同时保留核验清单和 R8 mapping 压缩文件。

最终该 APK 已通过 **68 个静态画面、96 次动画采样、13 个离线业务场景**，全部比较/断言成功。完整证据见 [核验清单](artifacts/performance-v210/verification.json)、[逐画面对比](artifacts/performance-v210/visual-comparison.json)、[动画检查](artifacts/performance-v210/animation-check.txt)、[原生业务检查](artifacts/performance-v210/native-checks.txt)、[源码摘要](artifacts/performance-v210/source-files.json) 和 [资源/效果溯源](artifacts/performance-v210/effect-provenance.json)。原始 68 张基线/候选图保留于本机 `output/performance-baseline/` 与 `output/performance-final-verified/`，未将大图和 APK 提交至 Git。

## 已实施

| 路径 | 改动与保持项 |
| --- | --- |
| 启动 | 提前提交相同的 768×768 徽标图片请求，复用同一图片缓存及原资源包回退；原启动条件、620ms 收退、320ms 淡出和支付素材准备条件不变。预热不保证一定在首帧完成。 |
| 首页和列表 | 商品筛选、分类、双列分组，以及市场、联系人、订单、委托、账单筛选/排序随真实数据和筛选条件更新；折叠进度不再使这些纯计算重复执行。保留原标题字体变化、列表 padding 和滚动范围。 |
| 加购/导航/开关 | 加购进度在原 graphicsLayer 内读取；导航将原两套弹簧收至指示器作用域；开关推迟至绘制/布局阶段读取，仍按原 Dp 位移取整。轨迹、时长、弹簧、触摸语义不变。 |
| 绘制 | 静态背景渐变复用，删除未使用的 rim 分配。原动态玻璃 shader、折射、模糊、倾斜和图片采样质量不变。 |
| 历史记录 | 用同排序键的有序插入/替换代替每插入一项就全量排序；相等时间保持原追加后稳定排序的次序，订单/委托继续保留 ID 次排序键。 |
| AI | 同条流式回复复用固定身份和时间元数据，时区/区域/助手名变化时重建；逐个原始事件更新正文、状态、来源，done 消息仍作权威覆盖。没有节流或改换 Markdown 渲染。 |
| 图片授权 | JSON 遍历移出界面读取所需的锁，锁内批次提交；保持同响应覆盖、保留期限继承和失败前已解析项，clear 的代数阻止旧批次复活。 |
| 请求身份 | 请求身份仍使用 commit 并等待落盘，在 IO 线程完成；主线程不被同步磁盘写入阻塞。由于 SharedPreferences 在 commit 返回前已更新内存，认证请求和恢复共用写入屏障；新增挂起点之前建立提交互斥。 |
| 商城读取 | 成功且实际重叠的分类/品牌/商品读取共享一次结果，购物袋仍逐调用刷新。失败、取消和之后的独立刷新不会被缓存吞掉。单页夹具实测两路重叠读取由原调用路径的 8 次降至 5 次。 |
| 通知偏好 | 同一响应使用一个 Editor 批量 apply；保留字段缺省值与异常前已解析部分的提交。 |
| 构建 | 新增非 debuggable、R8/资源压缩的 performance 候选，沿用现有安装身份和证书；新增应用 Profile 规则，合并 AndroidX Profile。Kotlin/Compose/Coil/Markwon 版本不变。 |

请求持久化失败的边界：旧实现忽略 commit=false，可能继续发送未耐久化的请求；现在返回 `PERSISTENCE_UNAVAILABLE` 并保留原序列化身份以供重试。这是保持“先落盘后发送”要求的安全边界，磁盘故障时错误反馈与旧实现不同，不能将其描述为所有故障行为逐字相同。取消后也保留原身份，账号变更时不发送旧账号请求。

## 同效证据

- 基线由已发布版本的同一 Android 源码重新构建，使用同一套固定夹具；不是声称直接在公网发布 APK 上植入了测试。基线重复采集的 68 张图自身零差异。
- 原基线 123 个文件中，101 个文件摘要保持不变；全部资源文件以及启动遮罩、折叠标题、支付体验、渐变玻璃头和 Markdown 渲染的原文件未改。22 个原文件修改及新增辅助文件可由分支差异追溯。
- 68 个固定画面覆盖浅/深色、1.0/1.4 字体缩放、首页 0/34/98/128dp 折叠、市场、信息、钱包、账单、订单、委托、玻璃、控件，以及加购 0/25/50/75/100% 进度。比较 PixelCopy 得到的应用窗口完整 RGBA 字节，包含 alpha；不比较系统栏。
- 动画测试将原源码控件与优化控件置于同一物理位置，在冻结的 Compose 时钟上切换可见层。两种主题、三次目标切换、八个时间采样、导航/开关两类控件，合计 96 次比较。采集之间时钟不得前进；没有用“肉眼接近”判定。
- 同位置基准校准后，debug 与最终 performance 目标分别完成 96 次动画比较，均零差异；派生列表的商品/分类/筛选/账单更新检查均通过。
- 149 项 JVM 测试通过，0 失败、0 错误；包含有序记录、图片授权并发/清理、请求持久化、AI 模板和重叠读取。
- 非 debuggable 的未压缩 profileCapture 目标已通过：持久化与真实请求链 4 场景、付款 4 场景、联系人 2 场景、图片缓存 3 场景。profileCapture 用于区分源代码行为和构建压缩问题。
- `lintDebug` 通过，0 errors、24 warnings。包括依赖/target 版本提示、原图标/存储建议；其中 5 条提示现有 Compose/Lifecycle 自定义 lint 要求更新的 lint API，因此不能把此次 lint 说成所有最新自定义规则均已运行。没有为消除提示升级运行库。

## 构建取舍与复现

AGP 8.6.1 原附带的 R8 无法正确处理当前 Kotlin 2.3 元数据；按照 [Android Kotlin 支持表](https://developer.android.com/build/kotlin-support) 将 R8 固定为 8.13.19，使用 [R8 官方覆盖方式](https://r8.googlesource.com/r8/+/refs/heads/main/README.md)。性能包保留测试与目标共享的公开运行接口，使独立测试 APK 能检查最终压缩产物；通常方法体继续优化，协程公开/受保护接口保留原可继承性以支持测试调度器。大量未使用图标仍可删除。这是保守的候选规则，未追求最小体积。测试夹具本身不打包进用户 APK。

应用 `baseline-prof.txt` 为依据已检查关键路径编写的规则，合并依赖自带 Profile。尝试从同进程 instrumentation 导出的 ART 配置为空，未将它伪称为设备采集的 Profile；包内实际 Profile 及设备安装结果另行核验。

最终 APK 包含 `baseline.prof` 11,018 字节与 `baseline.profm` 1,318 字节。ProfileInstaller 强制安装返回成功（result=1）；`cmd package compile -f -m speed-profile` 成功，设备 dexopt 报告目标为 `speed-profile / cmdline`。之后实际 Activity 冷启动 Status=ok、进程存活、crash buffer 为空。单次 `am start -W` 的 258ms 仅保留作启动冒烟日志，不代表动画完成时间或真实首页卡顿指标。见[安装与编译日志](artifacts/performance-v210/profile-install.txt)。

Windows 构建环境：JDK 17，SDK `C:/DeuteriumAPP/.tools/android-sdk`。在 `android-app` 执行：

```powershell
./gradlew.bat :ui-lab:testDebugUnitTest :ui-lab:lintDebug :ui-lab:assemblePerformance
./gradlew.bat :ui-lab:assemblePerformanceAndroidTest '-PdeuteriumTestBuildType=performance'
# Compose 时钟测试使用 AndroidJUnitRunner；必须安装与该 variant 映射一致的测试 APK。
./gradlew.bat :ui-lab:assemblePerformanceAndroidTest '-PdeuteriumTestBuildType=performance' '-PdeuteriumTestRunner=androidx.test.runner.AndroidJUnitRunner'
```

自定义 runner 的离线选项为 `performanceVisual`、`performanceIO`、`paymentTiming`、`contactsLayout`、`imageCache`；不得无参数启动该 runner，它的既有默认路径要求真实 QA 会话。此次夹具全部使用本地状态或独立拦截器，无生产账号和业务写入。

## 保留项与限制

联系人仍为原连贯卡片布局；行虚拟化涉及滚动锚点、头像首次呈现和卡片裁切，未在本轮替换。生命周期轮询、20MB 上传缓冲、登录会话加密以及逐帧标题测量仍是后续可测热点。本轮没有变更后台刷新间隔、启动等待条件、支付素材、传感器频率、图片分辨率或玻璃质量。

所有 Android 证据来自此次专用 API 35 模拟器（720×1280、240dpi）。没有用户手机的 FrameTimeline、真实账号首页数据或触摸手感验收，也没有测得真机启动/帧率改善百分比。固定场景像素一致与业务检查为本轮证据边界，不能推断所有机型、GPU、帧进度及线上故障均已穷尽。徽标预热缩短潜在准备路径，但未修改其可见时钟，无法据此承诺原设备顿挫已完全消失。
