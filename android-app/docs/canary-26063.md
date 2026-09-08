# Deuterium Android Canary 26063

## 定位

Canary 26063 是独立可共存的 Android 前端实验版，用于更激进地验证 UI、动画和滚动流畅度优化。

Canary 不修改后端通信接口，不修改 HTTP API、WebSocket 地址、消息类型、请求字段或响应字段。它复用 debug 环境的后端地址，仅通过 Android build type、应用包名和 `BuildConfig.CANARY_UI` 隔离前端实验。

## 构建信息

- Build type：`canary`
- 应用包名：`com.deuterium.app.canary`
- Launcher 名称：`Deuterium Canary`
- `versionCode`：`26063`
- `versionName`：`1.0.4-canary.26063`
- 构建命令：`.\gradlew.bat :app:assembleCanary`

Canary 与 Dev/Debug 的 `com.deuterium.app` 可同时安装。由于 Android 会按应用包名隔离数据，Canary 首次启动不会继承 Dev/Debug 已登录会话。

## Canary UI 实验

- 顶部栏显示 Canary 身份与 `26063` 版本标识，避免与 Dev 版混淆。
- 聊天气泡采用更接近即时通讯应用的非对称圆角，减少视觉噪声并提升消息流辨识度。
- 聊天列表新消息使用轻量 fade 与 placement 动画，保留 stable key 与 `contentType`，避免大量列表项重组。
- 底部导航、主要按钮和图标按钮使用低回弹 spring，减少高频点击时的弹性震荡。
- IME 出现/隐藏、输入框多行展开、临时提示显示隐藏在 Canary 中优先使用更短、更轻的动画。
- 聊天页增加当前时间线内搜索入口，搜索仅过滤前端已加载消息，不新增后端查询接口。
- 长按聊天消息打开底部操作面板，支持回复与复制；服务器事件只允许复制。
- 回复使用前端组合器实现，输入框展示引用 preview，发送时仍通过原 `content` 字段传输，不新增 WebSocket 字段。
- `@` 候选列表采用头像、在线状态点、关注/在线标识的紧凑列表样式，选择结果仍使用原 `mentionedPlayerRefs`。
- 聊天搜索 header 已从单个 pill 容器改为两段式 toolbar，避免搜索态标题、人数按钮、关闭按钮和搜索框互相挤压。
- 钱包页改为 Canary 专属的 TG-like 列表结构：顶部余额身份卡、圆形状态头像、胶囊操作按钮、流水行右侧金额。
- 转账页改为 Canary 专属的 TG-like 表单分组与底部 sheet：确认提交和反馈不再使用默认 AlertDialog。

## QA 边界

Canary 需要独立安装和启动验证：

```powershell
.\gradlew.bat :app:testDebugUnitTest
.\gradlew.bat :app:assembleCanary
```

安装后检查：

```powershell
adb shell pm list packages | Select-String "com.deuterium.app"
adb shell am start -n com.deuterium.app.canary/com.deuterium.app.MainActivity
```

如果没有在 Canary 中重新登录账号，只能验证登录页、安装元数据和基础启动状态；聊天、钱包和玩家目录的真实数据流 QA 需要在 Canary 包内重新登录后执行。
