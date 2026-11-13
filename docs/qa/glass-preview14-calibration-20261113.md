# 第十四版材质实测校准

2026-11-13。用户指出第十三版解决了交互问题，但底栏、输入栏和表情菜单的玻璃观感仍与酷安 / TG 相差明显。本版专门重新校准材质，保留第十三版头像防抖和分组规则。

已无线覆盖安装到原 Xiaomi 15，版本 `2.0.30-glass-preview.14 (22900)`，包名 `com.deuterium.app.uilab`，UID `10509` 保持。连接正常服务，未提交、推送、合并或公开发布。

## 参考实测

酷安 16.7.2 的原机无障碍布局显示玻璃外壳边界为 `[65,2420][1135,2602]`，density 3.25，对应 56 dp 高、20 dp 左右留白；标签容器为 48 dp 高，内外间距 4 dp。

捕获酷安同一页面滚动前后的截图，通过内容匹配获得 `-370 px` 位移，匹配相关度 `0.9999994`。将滚动后露出的内容映射回被底栏覆盖的位置，得到本机校准底图。该图片仅通过 ADB 提供给独立 QA 应用，没有打包进正式 App。

同内容对照还确认了玻璃以外的底部渐退衬底：在 y=2608..2630 区域，其效果近似约 0.73 的白色叠加；靠上逐渐减弱。旧版直接露出手势条附近的价格和图片，这一差异不是单独修改玻璃模糊能解决的。

TG 原机深色参考中，输入栏常见颜色约为 RGB `32/42/54`，面板约为 `37/47/62`，底部菜单约为 `31/39/50`。旧面板主要颜色为 `27/32/40`，偏暗且缺乏层次。

证据：[测量数据](C:/DeuteriumAPP/output/android-liquid-glass-preview/reference-study-20261113/coolapk-material-measurements.json)、[同底图对照](C:/DeuteriumAPP/output/android-liquid-glass-preview/preview-14-qa/nav-reference-comparison.jpg)。

## 改动

- 保持原底栏布局占位和四入口，将可见玻璃壳改为 56 dp、内胶囊 48 dp、4 dp 内边距；玻璃层的位置和左右留白按参考校准。普通无玻璃模式沿用原布局。
- 底栏模糊由 12 dp 降到 1.5 dp，着色 alpha 维持 .76；浅色改用纯白着色，恢复明亮的边缘高光。保留原阻尼形变、外溢采样、活动胶囊高光及阴影。
- 增加独立的 113 dp 底部渐退衬底。它只绘制在滚动内容上，不占布局空间，不参与玻璃自身的重复捕获，也不拦截触摸。
- 聊天输入栏可见高度由 48 dp 收到 44 dp，圆角 22 dp；保留原按钮、输入功能和横向排布。
- 深色输入栏着色为 `#263140`，大面板为 `#2C3749`，底部小胶囊为 `#20262F`，alpha 均为 .85。实际 QA 渲染的面板主色为 `37/47/62`，与参考对应区域一致。
- 输入栏和小菜单保持 6 dp 模糊，大面板保持 40 dp 磨砂通道；饱和度处理调整为 3。使用 0.5 dp 细描边，深色上沿 alpha .16、下沿 .08；浅色白色边缘，深色小控件不加黑色投影。
- 底部菜单改为 36 dp 高、18 dp 圆角、14 sp 标签，选中态改为低对比度材质内衬和中性文字。保留表情/贴纸功能和分类收起行为。
- 材质预设迁移版本为 2，只应用于底栏及聊天相关组，原浮层和顶部模糊参数保留。购物袋、更多入口、刷脸和其他页面布局未修改。

## 验证与边界

构建命令（工作目录 `android-app`）：

```text
gradlew.bat -I C:/DeuteriumAPP/output/android-liquid-glass-preview/preview-repositories.gradle :ui-lab:assembleChatMediaQa :ui-lab:assembleChatMediaQaAndroidTest :ui-lab:assemblePerformance :ui-lab:testPerformanceUnitTest -PdeuteriumTestBuildType=chatMediaQa --offline --console=plain
```

- 第十四版构建与 184 项单元测试通过，零失败。
- 使用与 Xiaomi 15 相同的 1200×2670、density 520 测试配置进行同底图校准。等待底图真实绘制后才截图，避免首帧空白样本。
- 浅深色底栏按压外溢、拖动、松手、取消，以及头像定位/点击/提及验证通过。
- 浅深色输入栏、表情面板、分类栏收起/恢复、末行避让、键盘切换、贴纸和图片预览回归通过。
- 深色面板主要背景像素与参考的对应色值一致；同底图的无图标材质采样区平均 RGB 绝对误差记录在 [校准指标](C:/DeuteriumAPP/output/android-liquid-glass-preview/preview-14-qa/calibration-metrics.json)。此指标只描述指定区域，不能当成整套界面的相似度或精确复刻证明。
- 原机已检查实际安装版本、UID、APK 哈希并查看商城画面，确认新外壳尺寸和底部渐退生效。

日志：[构建](C:/DeuteriumAPP/output/android-liquid-glass-preview/preview-14-final-build.log)、[同底图校准](C:/DeuteriumAPP/output/android-liquid-glass-preview/preview-14-calibration-verified.log)、[组件回归](C:/DeuteriumAPP/output/android-liquid-glass-preview/preview-14-reference.log)、[浅色聊天](C:/DeuteriumAPP/output/android-liquid-glass-preview/preview-14-chat-light.log)、[深色聊天](C:/DeuteriumAPP/output/android-liquid-glass-preview/preview-14-chat-dark.log)、[安装](C:/DeuteriumAPP/output/android-liquid-glass-preview/preview-14-phone-install.log)。

## 安装包

[Deuterium-2.0.30-glass-preview.14-22900.apk](C:/DeuteriumAPP/delivery/android-liquid-glass-preview/Deuterium-2.0.30-glass-preview.14-22900.apk)

SHA-256 `e41b0d3c187d25d0d01cc3e50e94722e60db1211853edf911a0ffb81283fb65b`，与手机内 APK 一致。签名仍为 `04a213066a78a968c2a1328a153a0967f99d117e0a826285e3f4af705db6f2f3`。沿用本机预览 versionCode 22900 约定，正常服务地址为 `https://47.103.99.34`。
