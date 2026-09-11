# 全局过度绘制真机检查与修正

2026-09-11。基于性能候选 `0492730`，工作区 `C:/DeiteriumAPP-IX/.worktrees/performance-audit-v210`，分支 `xiwanzi/global-overdraw-v210`。修正版已覆盖安装到本次连接的手机，账号与应用数据保留；版本仍为 **2.0.10（21000）**，未推送、合并或线上发布。

## 结论

全屏重度覆盖来自公共背景链路。原 [GlassAtmosphere](../../android-app/ui-lab/src/main/java/com/deuterium/app/uilab/LiquidGlass.kt) 每次绘制都执行一遍底色和三遍全屏渐变；外面还有根容器和窗口底色。`recordGlassBackdrop` 保存并回放绘制指令，并没有复用合成后的背景纹理。因此，不含卡片、文字或玻璃的空白区域也会显示为高次数覆盖。

修正后，商城、市场、信息、我的四页的普通背景由全屏红色变为蓝色；卡片、文字和玻璃按各自实际层次显示。底部玻璃仍为多层覆盖，模糊、折射、高光及动态前景保持原实现。Android 调试工具的色块含义参见[官方说明](https://developer.android.com/topic/performance/rendering/inspect-gpu-rendering)；计数着色与 GPU 耗时分别测量，未用着色结果代替时间数据。

## 实现

改动仅涉及两个主源码文件：

- [LiquidGlass.kt](../../android-app/ui-lab/src/main/java/com/deuterium/app/uilab/LiquidGlass.kt) 新增 `GlassScene`。尺寸或配色等缓存依赖变化时准备原始背景指令和屏幕背景缓存；正常绘制复用背景纹理，实时前景继续更新。
- 玻璃采样使用原始背景指令和当前前景，屏幕显示使用背景缓存和当前前景。这样保留玻璃原有的过滤与取整过程，不在玻璃前增加一次背景纹理采样。
- [DeuteriumActivity.kt](../../android-app/ui-lab/src/main/java/com/deuterium/app/uilab/DeuteriumActivity.kt) 接入该容器，并删除被内容完全覆盖的根 `Box` 底色；窗口底色、登录页和恢复状态的兜底保留。

三层渐变的颜色、透明度、位置、半径和顺序不变，原玻璃 shader、模糊、折射及高光参数不变。没有缓存整张动态页面，也没有降低图片分辨率或修改滚动、动画和业务代码。

在此手机分辨率下，一张 1200×2670 的 RGBA 背景原始像素量为 **12,816,000 字节，约 12.2 MiB**。这是换取减少重复绘制的纹理成本估计，不是实测新增总 GPU 内存；实际分配还包含对齐等开销。缓存随 Compose 图层生命周期释放，窗口尺寸或颜色变化会重新准备。

## 真机结果

设备为 Xiaomi 24129PN74C，API 37，1200×2670、520dpi，Skia Vulkan。安装前后 APK 均为非 debuggable、同包名、同版本、同签名。

四页分别检查左侧不含内容的背景条带：`x=8..23, y=1000..2199`，每页 19,200 像素。原版条带全部为红色着色，修正版全部退出红色着色；主色从 `[248,121,123]` 变为蓝色区域的 `[197,197,247]`。这是指定背景区域的检查结果，不是整页红色面积百分比。见[背景采样与截图摘要](artifacts/overdraw-v210/background-samples.json)。

关闭过度绘制着色后，在信息页先往返滑动预热一次，再使用同一套系统触摸命令往返四次。通过该窗口的 `FrameMetrics` 记录样本：

| 指标 | 原性能候选 | 本次修正版 |
| --- | ---: | ---: |
| 帧数 / 有效 GPU 样本 | 1,260 / 1,260 | 1,260 / 1,260 |
| GPU 耗时中位数 | 5.66 ms | 4.18 ms |
| GPU 耗时 P95 | 6.28 ms | 4.89 ms |
| 总帧耗时中位数 | 7.88 ms | 6.34 ms |
| 总帧耗时 P95 | 11.38 ms | 10.20 ms |

该组 GPU 中位数降低约 26.2%。这是此设备、此操作的一组对照，未做长期温度、频率与所有业务数据的控制，不能推断所有场景同幅度提升或不再掉帧。原始样本见[改前](artifacts/overdraw-v210/frames-before.json)、[改后](artifacts/overdraw-v210/frames-after.json)。

## 效果保持与验证

- 在同一手机上比较原绘制路径与最终 `GlassScene`：浅/深色各检查纯背景和玻璃场景，包含完整前景采样和仅背景采样，共 **4 组对照，0 个不同像素**。采用应用窗口 PixelCopy 的完整 RGBA8888 数据，不接受全黑/单色帧作证据。见[结果](artifacts/overdraw-v210/phone-pixels.json)。
- 将上一轮固定夹具改为使用最终 `GlassScene` 后，与原源码基线比较 **68 张画面，完整 RGBA 字节零差异**；覆盖多页面、浅深色、1.0/1.4 字体缩放、首页折叠、控件、玻璃和加购进度。此部分使用专用 API 35 模拟器，见[逐画面结果](artifacts/overdraw-v210/visual-comparison.json)。
- 149 项 JVM 测试通过，0 失败；`lintDebug` 通过，0 errors、24 warnings。现有 lint/API 和依赖版本提示仍保留，未为消除提示变更渲染库。
- 最终 APK 签名检查、真机覆盖安装及手机端文件 SHA-256 核对通过。原生实际页面采集使用 Android 平台接口，能检查原版与新版 R8 APK；像素夹具使用与候选匹配的测试 APK。

探索中，直接合并渐变 shader 在手机上产生了最多 3 级通道色差；普通背景缓存虽然背景本身一致，但玻璃仍有 1 级色差。两者未用于最终实现。最终保留原采样源的方案才通过零差异检查，见[探索数据](artifacts/overdraw-v210/experiments-phone.json)。

早期外部截图出现黑屏，已排除；自动采集曾因无障碍缓存读到旧页面而误报导航失败，已通过刷新缓存修正。最终四页采集均检查目标标题并完成。真实页面截图含用户内容，仅保留在本机忽略目录，未提交到 Git。

## 产物与复现

[修正版 APK](../../delivery/Deuterium-2.0.10-overdraw/Deuterium-2.0.10-21000-overdraw.apk) 为 8,875,538 字节，SHA-256：

```text
58633bc1490ab1294d7c836b818d02198eae9d2f70125a8257e6aa0e75f928c6
```

包名 `com.deuterium.app.uilab`，版本 2.0.10 / 21000；证书 SHA-256 与此前一致，为 `04a213066a78a968c2a1328a153a0967f99d117e0a826285e3f4af705db6f2f3`。APK 内仍包含 Baseline Profile，库、资源、版本和签名策略沿用上一轮。完整[核验清单](artifacts/overdraw-v210/verification.json)及 R8 mapping 压缩文件随本机产物保留。

JDK 17、SDK `C:/DeuteriumAPP/.tools/android-sdk`，在 `android-app` 构建：

```powershell
./gradlew.bat :ui-lab:assemblePerformance :ui-lab:testDebugUnitTest :ui-lab:lintDebug
./gradlew.bat :ui-lab:assemblePerformanceAndroidTest '-PdeuteriumTestBuildType=performance'
```

专用测试入口：`overdrawDiagnostics=verify` 做原绘制路径/修正路径像素对照；`overdrawPageCapture=before|after` 采集真实四页；`overdrawScrollBenchmark=before|after` 测量信息页滑动。必须记录实际 APK SHA，像素检查和耗时检查关闭过度绘制着色，计数截图开启；完成后恢复设备原值。不要无参数运行已有 `LiveBackendInstrumentation`，其旧默认路径使用真实业务 QA 会话。

本次未改账号、服务器配置、订单/资金逻辑或游戏插件，未清除应用数据。手机原过度绘制设置为 `show`，已恢复并保留开启，便于用户继续查看。

收尾已核对并移除手机上本轮生成的 33 个诊断文件，卸载本轮新装的测试组件，停止专用模拟器；修正版 App 保留在手机上。真实截图和原始诊断产物仍保留在本机 `output/overdraw-phone-v210/`，清理记录见[现场恢复](artifacts/overdraw-v210/cleanup.json)。
