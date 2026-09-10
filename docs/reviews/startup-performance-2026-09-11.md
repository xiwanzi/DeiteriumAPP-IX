# 启动与加载动画性能审查

2026-09-11 02:38：本轮修正已随 App/Web 2.0.10（21000）及配套 Go 发布，账号提醒已推送；见[实际部署记录](../deployment/deuterium-2.0.10-live.md)。下方“尚未部署/等待确认”为发布前快照。

2026-09-11，基于线上 2.0.9（20901），本机分支 `xiwanzi/commerce-refunds-performance`。以下修正尚未发布；未取得用户骁龙 8E 真机的帧轨迹，不能把代码热点直接等同于该设备卡顿的唯一原因。

## 已修正

| 热点与证据 | 修改 | 保留的行为 |
| --- | --- | --- |
| `AppLaunchOverlay.kt` 在组合阶段读取 `arrival.value` 构造 `offset(Dp)`，开屏徽标每帧位移会重新组合并布局 | 位移、缩放都在 `graphicsLayer` 内读取动画值 | 原位移、缩放、时长和启动就绪判断 |
| `AppIdentity.kt` 的内置徽标使用 `painterResource`；原 PNG 为 1254×1254、1,819,447 字节，会在主线程组合时首次解码 | 复用已有 Coil loader，以 768×768 目标尺寸异步请求；开屏与登录共用缓存 | 资源包徽标优先，文件缺失/解码失败回退内置图 |
| `GradientGlassHeader.kt` 的模糊层按整个 `backdrop.size` 记录，最终只显示顶部渐变区域 | 按顶部高度加 3 倍模糊半径的邻域采样，避免对其余整屏处理 | 连续渐隐及邻近像素采样；截图核对无截断接缝 |
| `WalletPage.kt` 在组合阶段取得刷新角度；`StoragePage.kt` 同样先把动画状态转换为整数 | 把状态读取推迟至图层旋转/Canvas 绘制 | 相同转速、指示形态、关闭动效后的静态状态 |
| `CouponAttention.kt` 的 getter 随每秒 `now` 更新重新创建未读列表，红点订阅者会被频繁通知 | 用 `derivedStateOf` 只在未读/到达内容发生变化时更新 | 到期及时消失、账号隔离、批次去重和离线确认 |
| 开屏 `PaymentSound.prepare` 在主线程创建声音池 | 改在 IO 调度器准备 | 付款确认声音、付款提交和未知结果恢复流程 |

顶部采样面积的减少是结构性变化，不能换算成同等比例的帧率提升。例如截图场景顶部 115dp、默认模糊 18dp，采样高度为 169dp（115 + 3×18），原先为整屏高度；实际 GPU 收益仍取决于设备和当前画面。

支付素材的前 22 帧矢量数组实际已为空，仅最后 8 帧有少量路径，JSON 总计 1786 字节。这部分检查后保留原实现，没有把它列为已修复的耗时来源。

## 验证与边界

- Android `:ui-lab:testDebugUnitTest :ui-lab:assembleDebug :ui-lab:assembleDebugAndroidTest` 成功，134 项单测通过。
- Android 35 x86_64 模拟器覆盖安装成功，桌面别名启动返回 `Status: ok`；单次 `am start -W` 的 `TotalTime=1862ms` 仅为启动冒烟证据，未做同条件前后基准，不能宣称加速比例。
- 原生通知脚本使用真实新通知卡片和新渐变头部，验证浅深色、大字、关闭动效、浮层避让、页面触摸、批次/账号隔离及到期消失。见[验收记录](../qa/commerce-refunds-performance.md)。
- 保留玻璃和动效开关，没有通过默认关闭效果来规避性能问题；没有改变真实资金提交时序。

## 后续最有价值的验证

当前 APK 的 `aapt dump badging` 明确包含 `application-debuggable`，构建路径也是 debug。工程未配置应用自己的 Baseline Profile 生成模块；依赖中包含 AndroidX profileinstaller，但不能据此认为本应用的启动和主要页面已经有专门的基线覆盖。

应在签名/升级兼容的正式构建上，使用目标手机分别采集冷启动、暖启动、首页前 10 秒、钱包刷新和列表滑动的 Perfetto/FrameTimeline，观察主线程、RenderThread、图片解码、JIT 与 GC，再决定是否增加应用 Baseline Profile。此项需要真机及正式构建验证，当前不报告为已经消除，也不根据单次模拟器启动数据作性能承诺。

依据：[Android 官方性能实作教程](https://developer.android.com/codelabs/jetpack-compose-performance)说明了主线程资源图片解码、把状态读取推迟到布局/绘制阶段以及正式构建测量的重要性；[Compose 阶段与性能](https://developer.android.com/develop/ui/compose/performance/phases)解释各阶段失效的范围。本轮具体发现来自上述本地代码，性能收益幅度仍待实测。
