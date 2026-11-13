# 聊天头像退场与上下渐退，第十六版

2026-11-13。已无线覆盖安装 Xiaomi 15：`com.deuterium.app.uilab`、`2.0.30-glass-preview.16 (22900)`，UID `10509` 保持，正常服务连接保留；未公开发布、推送或合并。

## 修正

- 第十五版将头像的绘制/退场边界从输入栏上缘扩展到消息绘制区域底边。组内吸附锚点仍在输入栏上方，但组头继续滚动时，头像会被推入输入栏后方，部分露出底边后才完全离开画面。保留第十三版直接读取列表布局的防抖修复。
- 消息与头像共用底部 48 dp 渐退。视觉压暗发生在原始内容采样之后，避免玻璃二次采样已压暗的像素，保留第十四版输入栏和表情面板的材质。
- 用户随后明确聊天顶部应使用与底部相同的逻辑。第十六版新增 `ChatTopFade`，与底部共同调用 `chatEdgeBrush`，仅反转渐变方向。聊天页不再使用通用 `GradientGlassHeader` 的模糊/调色路径；其他页面继续使用原通用头部。
- 过渡 alpha 从 0、96/285、176/285 到 232/285，浅色使用白色衬底，深色使用黑色衬底。聊天标题、按钮、输入框位置、分组规则以及底栏材质未改动。

## 验证

- 第十五版 185 项单元测试通过。头像逐帧检查 440 个绘制帧，最大定位误差 0 px，记录到 37 帧头像在输入栏下方、12 帧部分露出画面底边；覆盖往返滚动、分组边界、删除和历史插入。
- 第十五版深色聊天完整回归通过表情面板、键盘切换、图片/GIF、贴纸、附件抽屉和材质控件。
- 第十六版浅深色像素检查通过：顶部和底部镜像位置的亮度差不超过 2，浅色边缘亮度均为 208，深色均为 47。输入栏的高对比度采样结果保持 217 / 80，确认渐退没有被重复捕获进玻璃。
- APK 包名、版本、签名与原机哈希核对通过，已启动现有应用。第十六版仅改聊天顶部绘制路径，未重复执行未改动模块的全部测试。

记录：[第十五版构建](C:/DeuteriumAPP/output/android-liquid-glass-preview/preview-15-final-build.log)、[头像退场](C:/DeuteriumAPP/output/android-liquid-glass-preview/preview-15-avatar-frames-final.log)、[聊天回归](C:/DeuteriumAPP/output/android-liquid-glass-preview/preview-15-chat-dark-verified.log)、[第十六版构建](C:/DeuteriumAPP/output/android-liquid-glass-preview/preview-16-build.log)、[上下对称性及采样](C:/DeuteriumAPP/output/android-liquid-glass-preview/preview-16-edge-fade.log)、[安装](C:/DeuteriumAPP/output/android-liquid-glass-preview/preview-16-phone-install.log)。

验证中 Windows 的 SwiftShader 无头模拟器宿主发生访问异常，已切换宿主 GPU 并完成有效重跑。录屏探针还修正了将 LazyList 已附着但未摆放的预取节点误计为可见节点的判断；坐标误差和可见性仍通过实际绘制坐标检查。

## 构建与交付

在 `android-app` 使用原 Java 17 / Android SDK：

```text
gradlew.bat -I C:/DeuteriumAPP/output/android-liquid-glass-preview/preview-repositories.gradle :ui-lab:assemblePerformance :ui-lab:assembleChatMediaQa :ui-lab:assembleChatMediaQaAndroidTest -PdeuteriumTestBuildType=chatMediaQa --offline --console=plain
```

[Deuterium-2.0.30-glass-preview.16-22900.apk](C:/DeuteriumAPP/delivery/android-liquid-glass-preview/Deuterium-2.0.30-glass-preview.16-22900.apk)

APK SHA-256：`7acb197bb9bdfd54afd097c25dab20430127aa5c29a0253e07c1447357cc2aec`，与手机内 APK 一致。签名继续为 `04a213066a78a968c2a1328a153a0967f99d117e0a826285e3f4af705db6f2f3`。沿用本机预览 versionCode `22900` 约定。

参考行为来源为已核对的 [Telegram 13.2.2 ChatActivityFadeView](https://github.com/DrKLO/Telegram/blob/21f7d109da6ab3506c9e9577f98c6a38771ab335/TMessagesProj/src/main/java/org/telegram/ui/Components/chat/layouts/ChatActivityFadeView.java) 与 `BlurredBackgroundWithFadeDrawable`，采用项目自身的 Compose 绘制实现。
