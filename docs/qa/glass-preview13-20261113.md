# 第十三版玻璃预览验收

2026-11-13。已通过无线 ADB 覆盖安装至原 Xiaomi 15，包名 `com.deuterium.app.uilab`，版本 `2.0.30-glass-preview.13 (22900)`，原 UID `10509` 保持。正常服务地址 `https://47.103.99.34`。仅本机预览，未提交、推送、合并、修改线上更新清单或公开发布。

## 本轮实现

- 底栏继续使用 Kyant Backdrop 1.0.6 与 Catalog 的阻尼形变、双层标签采样。按原机酷安录像恢复外壳的柔和承托阴影，保留按压外溢、速度形变及活动胶囊阴影。取消拖动恢复原选项。
- 聊天输入栏、表情大面板、面板底部小胶囊分离材质和采样。输入栏采样聊天内容，小胶囊组合面板材质与表情网格，不重复采样聊天或捕获自身。
- 表情分类栏上滑收起并让出实际空间，反向滑动恢复；底部类别菜单浮在网格上，按滑动方向收起/返回。恢复连续滚动，取消此前强制整行吸附。保留原有类别、贴纸管理、导入和数量限制，本轮没有增加搜索或 GIF 新功能。
- 连续消息采用消息组头像，保留原头像栏宽度、头像资料点击、公共聊天长按提及。分组依据稳定玩家身份、收发方向、日期和相邻五分钟时间窗。
- 用户在第十二版反馈头像不流畅、抖动后，第十三版仅修头像跟随：移除逐行全局位置回调写回状态的反馈链；从 LazyList 当前布局读取组边界，使用绘制层直接更新头像位移；只准备可见组及邻近组头像。消息行不再另行执行独立位移动画，避免与头像跟随不同步。
- 用户随后明确“逻辑暂时不用管，主要是流畅度和防抖动”，因此第十三版保留第十二版的分组规则，不扩展其他消息规则。

商店两排工具栏、更多入口、搜索框与购物袋的 4 dp 间距、刷脸付款布局均保留。`ShopNavigation.kt`、`PaymentExperience.kt` 与本轮开始保存的第十一版工作副本逐字一致。资金请求、付款时序、消息发送和服务端均未修改。

## 材质校准值

以下是适配现有渲染器后的值；酷安业务代码经过加固，这些不是从酷安读取到的精确参数。

| 区域 | 模糊 dp | 着色 alpha | 折射量 / 宽度 dp | 高光（浅 / 深） |
| --- | --- | --- | --- | --- |
| 底栏外壳 | 12 | .76 | 16 / 16 | .48 |
| 聊天输入栏 | 6 | .85 | 8 / 9 | .46 / .14 |
| 表情大面板 | 40 | .85 | 8 / 24 | .34 / .12 |
| 表情底部胶囊 | 6 | .85 | 7 / 8 | .50 / .14 |

输入栏圆角 22 dp，面板顶部圆角 29 dp，小胶囊圆角 20 dp；深色聊天材质采用 `#20262F`，移除通用玻璃中的深色亮度/对比度压缩。小胶囊只在浅色使用约 3.7 dp 柔影。底栏外壳使用 24 dp 阴影半径、6 dp 垂直偏移，活动胶囊阴影随按压进度变化。

使用一次性 `referencePresetVersion=1` 迁移只更新底栏及聊天输入栏/面板组，原浮层和顶部模糊设置保留；之后用户手动调参不会在重启时被覆盖。

## 验证

最终构建命令（在本工作区 `android-app`，Java 17、原 Android SDK）：

```text
gradlew.bat -I C:/DeuteriumAPP/output/android-liquid-glass-preview/preview-repositories.gradle :ui-lab:assemblePerformance :ui-lab:testPerformanceUnitTest :ui-lab:assembleChatMediaQa :ui-lab:assembleChatMediaQaAndroidTest -PdeuteriumTestBuildType=chatMediaQa --offline --console=plain
```

- 第十三版构建与 184 项 JVM 测试通过，无失败或错误。
- 第十三版头像逐帧检查通过 440 个绘制帧，实际绘制位置与消息组边界定位的最大误差为 0 px。覆盖慢滑、快滑、反向、跨组、删除中间消息和历史前插，并检查可见头像遗漏/重叠。探针在 Compose 子内容绘制后执行，不把 Android `OnDraw` 中尚未完成的摆放状态误当作最终画面。
- 第十三版组件回归通过浅深色底栏外溢、拖动、松手、取消，以及组头像定位、资料点击和长按提及。
- 第十二版完成且第十三版未再修改的输入栏/表情面板：浅深色真实组件测试通过分类栏收起/恢复、浮动底栏、末行可见、表情插入/删除、贴纸、键盘切换、图片预览和附件抽屉。设置迁移、独立调参和重置检查通过。
- 第十二版原页面回归通过搜索、AI、委托筛选、转账表单、刷脸等待的浅深色检查。测试使用独立 `.chatmediaqa` 应用和本地夹具，未向真实账号发送消息或执行付款。
- Xiaomi 15 已核对第十三版版本号、UID、签名及手机内 APK SHA-256，并启动应用。第十二版曾实机查看材质、表情面板及分组头像；第十三版的逐帧数字来自隔离模拟器，不是原机 FPS 基准。

证据目录 [output/android-liquid-glass-preview](C:/DeuteriumAPP/output/android-liquid-glass-preview)：

- [最终构建](C:/DeuteriumAPP/output/android-liquid-glass-preview/preview-13-final-build.log)
- [头像逐帧检查](C:/DeuteriumAPP/output/android-liquid-glass-preview/preview-13-avatar-frames-final.log)
- [第十三版组件回归](C:/DeuteriumAPP/output/android-liquid-glass-preview/preview-13-reference-final.log)
- [浅色聊天回归](C:/DeuteriumAPP/output/android-liquid-glass-preview/preview-12-chat-light-final.log)
- [深色聊天回归](C:/DeuteriumAPP/output/android-liquid-glass-preview/preview-12-chat-dark.log)
- [原页面回归](C:/DeuteriumAPP/output/android-liquid-glass-preview/preview-12-pages.log)
- [原机安装结果](C:/DeuteriumAPP/output/android-liquid-glass-preview/preview-13-phone-install.log)

手机拒绝安装新增的独立 QA 包，原应用覆盖更新成功，因此没有要求用户继续调整安装开关。早期模拟器测试受到系统窗口手势和无障碍缓存影响，已在独立模拟器重启、修正测试注入方式和缓存读取后完成有效重跑；以上只列最终通过的结果。

## 交付

[Deuterium-2.0.30-glass-preview.13-22900.apk](C:/DeuteriumAPP/delivery/android-liquid-glass-preview/Deuterium-2.0.30-glass-preview.13-22900.apk)

- APK SHA-256 `c27db41aecfc5bb4fcbce0be29fd5cc737aa4484807b6a4be51dfa2b134f6edf`，与手机内 APK 一致。
- 签名 SHA-256 `04a213066a78a968c2a1328a153a0967f99d117e0a826285e3f4af705db6f2f3`，沿用现有应用签名。
- 保留本机预览的 versionCode `22900` 约定，避免破坏与当前正式基线的覆盖回退关系。版本名区分迭代。

本机对照研究来源见 [酷安 / Telegram 实查](coolapk-telegram-glass-reference-20261113.md)。Telegram 源码仅作行为和参数研究，未将其 GPL 实现复制进应用；实际材质继续使用原 Kyant 组件。
