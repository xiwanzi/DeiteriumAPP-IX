# Xiaomi 15 酷安 / Telegram 玻璃与聊天交互实查

2026-11-13。本次完成无线连接、原机 APK 提取、录屏、用户视频分帧与对应版本源码核对。属于参考研究，未修改 App 实现、未安装新预览、未发布。手机上的 Deuterium 仍为 `2.0.30-glass-preview.11 (22900)`。

## 设备与版本证据

无线 ADB 已核对为原 Xiaomi 15（`24129PN74C` / `dada`），1200 × 2670，density 520。用户给出的 VPN 接口不能直接作为本机访问地址，使用同一手机的局域网接口与当前调试端口连接。

| 应用 | 包名 | 原机版本 | 提取结果 |
| --- | --- | --- | --- |
| 酷安 | `com.coolapk.market` | `16.7.2 (2611101)` | `base.apk`，113,560,383 bytes |
| Telegram | `org.telegram.messenger` | `13.2.2 (73031)` | base + arm64-v8a + xxhdpi 三个 APK，完整保留 split |

本地四个 APK 的 SHA-256 均与手机 `sha256sum` 一致。清单见 [apk-manifest.json](C:/DeuteriumAPP/output/android-liquid-glass-preview/reference-study-20261113/apk-manifest.json)，安装包见 [提取目录](C:/DeuteriumAPP/output/android-liquid-glass-preview/reference-study-20261113/apks)。仅提取安装代码和资源，未提取账号数据库或登录凭据。

Telegram 官方源码固定为 `DrKLO/Telegram@21f7d109da6ab3506c9e9577f98c6a38771ab335`，其 `gradle.properties` 为 `APP_VERSION_NAME=13.2.2`、`APP_VERSION_CODE=7303`。APK 中 `res/raw/liquid_glass_shader.agsl` 与该提交源码在归一化换行后逐字一致，见 [着色器对照](C:/DeuteriumAPP/output/android-liquid-glass-preview/reference-study-20261113/telegram-shader-match.json)。这证明版本及该渲染资源对应，不代表已完成整个 APK 的可复现构建验证。

## 酷安底栏

实际查看并录制了首页静止、按住首页、拖向数码、松手、拖回首页。见 [动作录屏](C:/DeuteriumAPP/output/android-liquid-glass-preview/reference-study-20261113/coolapk-nav-motion.mp4)和[状态对照](C:/DeuteriumAPP/output/android-liquid-glass-preview/reference-study-20261113/coolapk-nav-comparison.jpg)。

- 静止时外壳保持轻薄，选中区域为低对比度胶囊，完整包含图标和文字。
- 按住时外壳轻微整体扩张，内胶囊明显膨胀并越过外壳轮廓。溢出区域仍有页面内容折射，背景捕获不能只限于原底栏矩形。
- 拖动时选中标签与胶囊联动，位移与形变有不同步的弹性响应。松手后的归位和缩小是连续过渡。
- 外壳周围有柔和、范围较宽的承托阴影，活动胶囊边缘还有随按压变化的高光/阴影。不能把所有阴影清零，也不能对每个普通按钮统一加同样的投影。

APK 可见的模块证据为 `META-INF/Glass:backdrop.kotlin_module`、`liquidglass.kotlin_module`、`liquidglass-coolapk.kotlin_module`。后者相关元数据保留 `com.coolapk.market.widget.compose.liquid.LiquidGlassTabPanelKt` 和 `internal.DragGestureInspectorKt`，Compose UI 版本资源为 `1.12.0`。这支持“Compose + Backdrop 类玻璃组件”的判断。

可直接读取的 DEX 只有 30 个网易加固包装类，未取得业务渲染方法。因此本次没有把酷安的具体 Backdrop 版本、弹簧系数、模糊半径或阴影数值认定为已证实。没有使用其他人的反编译报告替代原机版本证据，也没有找到可确认属于酷安官方这版底栏的开源源码。

我们当前已使用 Kyant Catalog 的双层标签背景、真实页面采样、活动透镜与阻尼拖动。实际差别至少包括外壳 `shadow = null`，以及后续迭代加入的材质和深色亮度/对比度压缩。后续应保留现有四入口与布局，分别校准外壳、活动胶囊、标签层和按压状态，按原机录屏验收，不能再用“库相同”代替效果一致。

## Telegram 的材质分工

Telegram 13.2.2 的相关实现是自有 Android View / RenderNode / RenderEffect / RuntimeShader 管线，不能把它视为 Kyant Compose 组件。相同的是实时背景采样与 AGSL 折射这一类底层能力。

1. 聊天输入栏从聊天内容与壁纸组合的背景捕获中取样。
2. 表情面板的大背景使用较强的磨砂通道，避免消息文字在大面积背景上形成脏斑。
3. 面板底部的“表情符号 / GIF / 贴纸”胶囊与删除按钮，单独捕获表情网格。因此能看到背后表情颜色透进小控件，而不是再次折射聊天消息。
4. 阴影、描边和颜色按主题处理。普通深色 provider 的黑色阴影为零，上下描边分别使用白色 alpha `0x28` / `0x14`；浅色保留柔影。compact 模式有独立覆盖规则，不能混用两套参数。

### 可核实的参数与用途

这些是 TG 源码值，不是已写入 Deuterium 的配置。两套渲染器的参数含义不同，不能逐项按同名照抄。

| 场景 | TG 13.2.2 源码值 |
| --- | --- |
| 普通液态玻璃背景通道 | 屏幕空间 6 dp 模糊，4 倍降采样，额外饱和度处理 |
| 强磨砂通道 | 额外约 `40 - 1.66` dp 模糊、8 倍降采样，接在已有采样结果之后 |
| 普通透镜 | intensity `0.75`，index `1.5`，默认厚度 `11 dp`，实际厚度受短边的 `1/5` 限制 |
| 输入栏 | 圆角 `22 dp`，drawable padding `7 dp`；浅色背景 alpha 为 `216/255` |
| 表情 / 内置键盘大背景 | 顶部圆角 `29 dp`；厚度请求 `32 dp`，intensity `0.4`，仍经过上述尺寸钳制 |
| 表情底部类别胶囊 / 小按钮 | 圆角 `18 dp`，padding `6 dp`，液态模式背景 alpha `0.85`；描边 `0.5 dp` |
| 上述小按钮浅色投影 | 颜色 `0x40000000`，radius `11/3 dp`，dy `2/3 dp`；普通深色无该投影 |

关键源码：[ChatActivity 背景分工](https://github.com/DrKLO/Telegram/blob/21f7d109da6ab3506c9e9577f98c6a38771ab335/TMessagesProj/src/main/java/org/telegram/ui/ChatActivity.java#L2597)、[ChatInputViewsContainer](https://github.com/DrKLO/Telegram/blob/21f7d109da6ab3506c9e9577f98c6a38771ab335/TMessagesProj/src/main/java/org/telegram/ui/Components/chat/ChatInputViewsContainer.java#L77)、[EmojiView 采样源](https://github.com/DrKLO/Telegram/blob/21f7d109da6ab3506c9e9577f98c6a38771ab335/TMessagesProj/src/main/java/org/telegram/ui/Components/EmojiView.java#L2846)、[主题 provider](https://github.com/DrKLO/Telegram/blob/21f7d109da6ab3506c9e9577f98c6a38771ab335/TMessagesProj/src/main/java/org/telegram/ui/Components/blur3/drawable/color/BlurredBackgroundColorProviderThemed.java#L103)、[小按钮参数](https://github.com/DrKLO/Telegram/blob/21f7d109da6ab3506c9e9577f98c6a38771ab335/TMessagesProj/src/main/java/org/telegram/ui/Components/blur3/drawable/color/impl/BlurredBackgroundProviderImpl.java#L63)。

## 表情菜单的滚动方式

原机确认了完整过程，见 [录屏](C:/DeuteriumAPP/output/android-liquid-glass-preview/reference-study-20261113/tg-panel-motion.mp4)和[四状态对照](C:/DeuteriumAPP/output/android-liquid-glass-preview/reference-study-20261113/tg-panel-comparison.jpg)。

- 最初显示面板分类栏和搜索栏。
- 向上滑动，搜索栏先随列表上移；继续滑动后分类栏也移出，网格使用腾出的空间。
- 隐藏的是表情面板内的搜索和分类栏，聊天标题与输入栏保持原有位置。
- 反向滑动可重新带出分类栏，继续返回顶部才完整恢复搜索栏。
- 底部类别胶囊与删除按钮也会根据方向累计位移收起/返回。它们浮在网格之上，不占固定的实色横条。
- 允许连续自由滚动。吸附的是半展开的栏，不是强制让全部表情按完整行停靠。原机画面在边界可出现裁切中的半行，这是正常连续滚动的一部分。

代码在 `EmojiView.checkEmojiSearchFieldScroll` / `checkEmojiTabY` / `animateTabsY` / `checkBottomTabScroll`。普通表情分类栏最大位移 `36 dp`，贴纸栏 `48 dp`；停下后按 `0.5` 比例阈值选择展开/隐藏，过渡 `200 ms`。搜索激活状态另有规则，不能用滚动把正在输入的搜索框直接隐藏。

我们当前 `ChatExpressionPanel` 是固定 `44 dp` 分类行 + 网格 + 固定 `52 dp` 底行，且 `ExpressionGrid` 强制整行吸附。对照结果表明，要获得用户所说的连贯感，需调整面板内部的滚动消耗、栏的位移/占位和采样层；只把顶栏 alpha 改为零仍会留下空白和割裂。

来源：[分类栏与吸附](https://github.com/DrKLO/Telegram/blob/21f7d109da6ab3506c9e9577f98c6a38771ab335/TMessagesProj/src/main/java/org/telegram/ui/Components/EmojiView.java#L5278)、[搜索随动](https://github.com/DrKLO/Telegram/blob/21f7d109da6ab3506c9e9577f98c6a38771ab335/TMessagesProj/src/main/java/org/telegram/ui/Components/EmojiView.java#L5444)、[底部类别栏](https://github.com/DrKLO/Telegram/blob/21f7d109da6ab3506c9e9577f98c6a38771ab335/TMessagesProj/src/main/java/org/telegram/ui/Components/EmojiView.java#L5163)。

## 连续消息头像与碰撞

用户视频已按时间分帧，见 [时间轴图](C:/DeuteriumAPP/output/android-liquid-glass-preview/reference-study-20261113/user-video-contact-sheet.jpg)。原机打开同一群聊也观察到连续两条消息共用一个头像。补充头像录屏因手机切到其他应用被前台保护中止，没有把不存在的录像计作证据。

TG 先按发送者、收发方向、相邻时间等条件连接消息组。常规相邻时间阈值为五分钟，日期分隔、发送者变化、特定服务/付费/转发消息会打断或改变分组。头像绘制不固定在每一行的顶部。

绘制时先找到消息组下边缘，把头像约束在可见列表下边界以内；然后回溯同组上边界，限制头像不得越过组头。相邻组接近时，由各自边界约束产生推挤感。同一组已存在后续可见单元时，前面的重复头像不再绘制。这个效果主要来自几何约束，而非给每个头像加弹簧抖动。

实现参考为 `ChatActivity` 的 `drawPinnedBottom/drawPinnedTop` 分支、`maxY` 限制、`y - dp(48) < top` 和 `setImageY(y - dp(44))`。我们当前 `ConversationPages` 每条非本人消息直接绘制 `PlayerAvatar`，需以稳定玩家身份和消息 ID 建立展示分组，并在现有头像栏内做组级定位。

来源：[分组条件](https://github.com/DrKLO/Telegram/blob/21f7d109da6ab3506c9e9577f98c6a38771ab335/TMessagesProj/src/main/java/org/telegram/ui/ChatActivity.java#L38630)、[头像绘制与边界](https://github.com/DrKLO/Telegram/blob/21f7d109da6ab3506c9e9577f98c6a38771ab335/TMessagesProj/src/main/java/org/telegram/ui/ChatActivity.java#L6198)。

## 接入现有 App 的具体边界

| 改动点 | 当前实现入口 | 应验证的行为 |
| --- | --- | --- |
| 底栏外壳承托、按压形变与阴影 | `MovingGlassNav` / vendor `LiquidBottomTabs` | 保留四入口位置；胶囊包含图标文字；外溢采样有效；快速反向、取消拖动和减弱动态正常 |
| 输入栏 / 大面板 / 类别胶囊分离采样 | `ChatGlass` / `MessageComposer` / `ChatExpressionPanel` | 输入栏采样聊天，类别胶囊采样网格；深色无发黑脏斑；不循环捕获自身玻璃 |
| 搜索与分类的联动收起 | `ChatExpressionPanel` | 上滑让出真实空间；反向恢复；搜索聚焦保护；IME、面板切换连续；保留贴纸管理和导入入口 |
| 消息组头像 | `ConversationPages` | 两条/长组/日期切换/图片混合/删除/历史追加时只显示正确组头像；碰撞连续；头像点击及长按提及保持 |

商店两排工具栏、更多按钮、购物袋与搜索框的 4 dp 间隔、刷脸和付款布局不属于这些参考点，不应借此重新排版。本次新要求覆盖此前“表情网格一律整行停靠”的临时修补方案，但不意味着任意改动其他页面。

源码副本只保存在本机研究目录 `telegram-source`，附官方 LICENSE。Telegram Android 官方说明为 GPL v2 or later；后续若复用具体源代码，需要按许可证处理。当前应用仍使用 Kyant Apache-2.0 组件，本次未将 TG 代码混入应用模块。

## 验证范围

已完成原机包名/版本/设备核对、四 APK 哈希复核、酷安浅色底栏动态录屏、TG 深色表情面板动态录屏、用户头像视频逐段观察、版本对应与 shader 内容核对。未执行真实发送、购买、付款或账号更改。没有声称本次已完成酷安深色或 TG 浅色的原机验证；没有构建或推送第十二版。
