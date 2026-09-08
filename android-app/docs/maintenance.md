# DeuteriumAPP Android 前端维护文档

本文面向后续 Android 前端维护者，说明当前 `android-app` 的工程结构、运行配置、核心数据流、维护边界和常见改动方式。本文只覆盖 Android 前端；后端 API、Minecraft 插件桥和服务端经济系统以 `docs/contracts/` 与后端仓库实现为准。

## 1. 项目概览

- 产品形态：原生 Android App。
- 技术栈：Kotlin、Jetpack Compose、Material 3、Retrofit、OkHttp、Gson、SQLiteOpenHelper。
- 包名：`com.deuterium.app`。
- 当前版本：`versionCode 6` / `versionName 1.0.4`。
- 最低版本：`minSdk 26`。
- 目标版本：`targetSdk 35`。
- 工程形态：单 Activity、单 app module，目前没有拆 Gradle 多模块。
- 主要功能：
  - 账号注册、登录、修改密码。
  - 钱包余额、刷新余额、最近流水、玩家转账。
  - App 与 Minecraft 服务器公共聊天互通。
  - 本机聊天历史保存、搜索、删除。

当前前端不直接连接数据库，不直接修改游戏经济数据，不构造或解析服务器身份 UUID。玩家身份引用使用后端返回的 opaque `playerRef`，仅作为接口参数保存和提交，不展示给用户。

## 2. 关键目录和文件

```text
android-app/
  app/build.gradle.kts
  app/src/main/AndroidManifest.xml
  app/src/main/res/xml/data_extraction_rules.xml
  app/src/main/java/com/deuterium/app/MainActivity.kt
  app/src/main/java/com/deuterium/app/data/ApiModels.kt
  app/src/main/java/com/deuterium/app/network/ApiClient.kt
  app/src/main/java/com/deuterium/app/network/BackendApi.kt
  app/src/main/java/com/deuterium/app/repository/AppRepositories.kt
  app/src/main/java/com/deuterium/app/repository/ChatFeedMerger.kt
  app/src/main/java/com/deuterium/app/repository/ChatHistoryStore.kt
  app/src/main/java/com/deuterium/app/ui/theme/Theme.kt
  app/src/test/java/com/deuterium/app/repository/ChatFeedMergerTest.kt
```

职责划分：

- `MainActivity.kt`：Compose UI、页面导航、表单交互、用户操作入口。
- `ApiModels.kt`：后端统一响应、业务请求体、业务数据模型、UI 内部聊天历史模型。
- `ApiClient.kt`：OkHttp、Retrofit、统一错误解析、WebSocket 创建、环境地址。
- `BackendApi.kt`：HTTP 接口定义。
- `AppRepositories.kt`：账号、钱包、聊天仓库；负责调用网络层、维护 UI 状态、处理 WebSocket 消息。
- `ChatFeedMerger.kt`：聊天消息按 `messageId` 去重合并、保持时间顺序，用于 WebSocket 与最近 100 条补齐共用。
- `ChatHistoryStore.kt`：本地 SQLite 聊天历史保存、搜索和删除。
- `Theme.kt`：Material 3 主题、颜色预设、动态取色。
- `ChatFeedMergerTest.kt`：覆盖聊天补齐插入、重复消息去重、HTTP 补齐不删除服务器事件。

## 3. 构建环境与命令

项目当前使用 Gradle Wrapper：

```powershell
cd C:\DeuteriumAPP\android-app
.\gradlew.bat :app:testDebugUnitTest
.\gradlew.bat :app:assembleDebug
.\gradlew.bat :app:assembleRelease
```

APK 输出：

```text
android-app/app/build/outputs/apk/debug/app-debug.apk
android-app/app/build/outputs/apk/release/app-release-unsigned.apk
```

Debug 包是 Android Debug 签名，可用于本机或小范围测试，不适合作为正式分发包。Release 当前产物为 unsigned APK，正式发布前必须补正式签名配置。

当前 debug 安装包版本为 `versionCode 6` / `versionName 1.0.4`，路径为：

```text
android-app/app/build/outputs/apk/debug/app-debug.apk
```

常用 ADB 安装命令：

```powershell
C:\DeuteriumAPP\.tools\android-sdk\platform-tools\adb.exe install -r C:\DeuteriumAPP\android-app\app\build\outputs\apk\debug\app-debug.apk
```

## 4. 环境地址与明文流量

环境地址在 `app/build.gradle.kts` 的 `buildTypes` 中通过 `BuildConfig` 注入。

Debug 当前用于内网穿透联调：

```text
HTTP_BASE_URL = http://deuterium.s.odn.cc:80/api/v1/
CHAT_WS_URL   = ws://deuterium.s.odn.cc:80/api/v1/chat/ws
usesCleartextTraffic = true
```

Release 当前用于生产配置：

```text
HTTP_BASE_URL = https://deuterium.s.odn.cc/api/v1/
CHAT_WS_URL   = wss://deuterium.s.odn.cc/api/v1/chat/ws
usesCleartextTraffic = false
```

维护原则：

- 不要把 HTTP/80 联调配置带入 release。
- 如果后端 HTTPS 修好，只需要调整 debug 的 `BuildConfig` 地址即可。
- 正式上线必须使用 HTTPS/WSS，并配置可被 Android 信任的证书链。

## 5. 后端接口约定

HTTP 接口统一通过 `BackendApi` 调用。`ApiClient.request` 负责把后端统一响应转换为 `RepoResult`。

统一响应形态：

```json
{
  "requestId": "req_opaque",
  "data": {},
  "page": {},
  "error": {
    "code": "ERROR_CODE",
    "message": "用户可读错误",
    "retryAfterSeconds": 10
  }
}
```

鉴权规则：

- 登录后 token 保存在普通 `SharedPreferences`。
- 需要登录态的 HTTP 请求由 OkHttp 自动添加 `Authorization: Bearer <token>`。
- 以下未登录接口不添加 Authorization：
  - `POST /account/registration-code`
  - `POST /account/register`
  - `POST /account/login`
  - `POST /account/password-reset-code`
  - `POST /account/password-reset`
- WebSocket 使用 Authorization header 鉴权。

错误处理：

- 后端业务错误优先展示 `error.message`。
- 超时、网络异常、非 JSON 响应统一转成中文提示。
- WebSocket 鉴权失败会清理本地登录态并回到登录页。

## 6. 会话与本地数据

`AppPreferences` 实现 `SessionStore`，保存：

- token
- userId
- playerRef
- gameId
- qq
- identityStatus
- 外观设置
- 是否显示服务器事件

安全注意：

- token 按当前决策使用普通 `SharedPreferences`，没有使用加密存储。
- `AndroidManifest.xml` 设置 `android:allowBackup="false"`。
- `data_extraction_rules.xml` 排除了会话偏好和聊天历史数据库，避免系统备份带走本地敏感数据。

退出登录：

- 调用 `/account/logout`。
- 无论后端是否成功，都会清理本地 token 和用户资料。

修改密码：

- 走游戏内验证码流程。
- 成功后清理本地登录态，引导重新登录。

## 7. 页面结构与导航

当前只有一个 `MainActivity`。

登录前：

- `AuthScreen`
- `LoginForm`
- `RegisterForm`
- `ResetPasswordForm`

登录后底部导航：

- `WalletScreen`：钱包和最近流水。
- `TransferScreen`：转账。
- `ChatScreen`：公共聊天、在线玩家、提及玩家。
- `ProfileScreen`：我的资料、账号安全、外观设置、聊天设置、聊天记录入口、退出登录。

`MainSection.Transfer` 不在底部导航中直接展示，由钱包页或聊天在线玩家入口进入。

## 8. 账号流程

注册：

1. 用户输入游戏内 ID、QQ、密码。
2. 调用 `POST /account/registration-code`。
3. 后端返回 `verificationToken`。
4. 前端只提示去游戏内查看验证码，不显示验证码明文。
5. 用户输入验证码后调用 `POST /account/register`。
6. 成功后保存 token 和 `UserProfile`。

登录：

1. 用户输入玩家 ID 或 QQ + 密码。
2. 调用 `POST /account/login`。
3. 成功后保存 token 和 `UserProfile`。

恢复登录态：

1. App 启动检查本地 token。
2. 有 token 时调用 `GET /account/me`。
3. 成功更新本地用户资料；未授权则清理本地登录态。

修改密码：

1. 调用 `POST /account/password-reset-code`。
2. 保存后端返回的 `verificationToken`。
3. 提交验证码和新密码到 `POST /account/password-reset`。
4. 成功后清理登录态。

## 9. 钱包与转账

钱包页进入时：

- 调用 `GET /wallet/balance` 获取余额。
- 调用 `GET /wallet/records?limit=20` 获取最近流水。

刷新余额：

- 调用 `POST /wallet/balance/refresh`。

收款方搜索：

- 调用 `GET /wallet/recipients/search?query=...&type=auto`。
- 前端保存和提交 `playerRef`，不展示、不解析、不构造 UUID。

转账：

- 用户确认提交时生成一次性 `clientRequestId`。
- 调用 `POST /wallet/transfers`。
- 支持 `processing`、`success`、`failed`、`unknown` 状态。
- `unknown` 不自动重试，提示用户稍后查余额和流水。

服务器内 `/pay` 流水：

- 前端只展示 `/wallet/records` 返回的数据。
- 若服务器内 Pay 需要出现在最近流水，后端需要把 Pay 记录并入 `/wallet/records`。
- 若未来需要区分来源，建议后端契约新增 `source` 或 `type` 字段，例如 `app_transfer`、`server_pay`。

## 10. 聊天流程

进入聊天页时：

1. `GET /chat/messages?limit=100`
2. `GET /chat/presence`
3. `GET /chat/online-players`
4. 建立 WebSocket：

```text
{CHAT_WS_URL}?includeServerEvents=true|false
```

发送消息：

- 使用 WebSocket 消息 `type = chat.send`。
- 前端只提交 `clientMessageId` 和 `content`。
- 不提交 sender、gameId、UUID。

接收消息：

- `chat.message`：加入聊天列表，写入本地历史。
- `server.event`：清理 Minecraft 颜色/格式码后加入列表，写入本地历史。
- `presence.update`：更新在线人数和在线玩家列表。
- `chat.send.result`：失败时显示错误。
- `error`：显示后端返回错误。

重连体验：

- WebSocket 断开后仍自动重连。
- WebSocket 打开后会立即补齐最近 100 条公共聊天，覆盖断线期间的新消息。
- 聊天页前台可见期间每 15 秒调用 `GET /chat/messages?limit=100` 做一次静默补齐。
- App 从后台回到前台时会重新建立聊天 WebSocket，并立即补齐最近 100 条公共聊天。
- “连接中断 / 正在重连”这类非阻断提示不显示。
- 发送失败、登录失效等影响用户操作的提示仍显示。
- 如果断线期间服务器产生超过最近 100 条消息，前端在不改后端接口的前提下只能补回最近 100 条范围内的消息。

自动滚动：

- 聊天列表使用 `rememberLazyListState`。
- 用户停在底部时，新消息自动滚到最新。
- 用户向上翻历史时不自动跳到底部，而是显示“X 条新消息”提示。
- 用户点击“X 条新消息”提示或回到底部后恢复自动跟随。

服务器格式码清理：

- 当前只清理 `server.event` 的内容。
- 移除 Minecraft legacy 颜色/格式码：`&0-9`、`&a-f`、`&k-o`、`&r` 以及 `§` 开头的同类格式码。
- 玩家普通聊天不做清理，避免误删正常输入的 `&` 字符。

## 11. 本地聊天历史

本地聊天历史由 `ChatHistoryStore` 管理，使用 SQLiteOpenHelper，不依赖 Room。

数据库：

```text
deuterium_chat_history.db
```

表：

```text
chat_messages
```

核心字段：

- account_id：当前 App 用户 ID，用于账号隔离。
- message_id：后端消息 ID 或服务器事件 ID。
- sender：显示用发送者。
- content：显示用内容。
- kind：消息类型或事件类型。
- sent_at：消息发生时间。
- display_time：UI 显示时间。
- mine：是否当前用户发送。
- event：是否服务器事件。
- received_at：本机保存时间。

写入策略：

- 最近 100 条聊天加载成功后批量写入。
- WebSocket 新聊天实时写入。
- WebSocket 服务器事件实时写入。
- 主键为 `(account_id, message_id)`，重复消息会覆盖，不会重复插入。

搜索：

- “我的”页进入“聊天记录”。
- 空关键词展示最近本地历史。
- 非空关键词按 `sender` 或 `content` 模糊搜索。
- 当前默认返回最多 200 条结果，避免一次性渲染过多 UI 项。

删除：

- “删除当前账号的所有聊天记录”只删除当前登录账号在本机保存的历史。
- 不删除后端消息。
- 不影响当前在线聊天 WebSocket 流。

## 12. UI 与文案维护规则

- 不展示 UUID、serverUuid、playerRef、身份权威等内部概念。
- 面向用户的身份文案统一为“玩家身份：已绑定服务器身份”这类可理解表达。
- 余额和流水金额显示为“信用点”语义，内部保留后端 decimal string，不转浮点数做业务判断。
- Debug、原型、模拟失败、测试验证码等文案不得进入生产 UI。
- 图标优先使用 Material Icons；当前项目已引入 `material-icons-extended-android`。
- 页面保持当前 Compose + Material 3 风格，不在维护任务中重做视觉设计。

## 13. 常见维护任务

新增 HTTP 接口：

1. 在 `ApiModels.kt` 增加请求/响应模型。
2. 在 `BackendApi.kt` 增加 Retrofit 方法。
3. 在对应 Repository 中调用 `apiClient.request { ... }`。
4. UI 只消费 `RepoResult` 或 Repository 状态，不直接处理 Retrofit response。

新增 WebSocket 消息类型：

1. 在 `handleSocketMessage` 中按 `type` 增加分支。
2. 只信任后端提供的身份字段，不让前端构造发送者身份。
3. 需要进入历史的消息转换为 `ChatFeedItem` 后调用 `saveHistory`。
4. 对非阻断连接状态，避免在聊天页显示频繁提示。

新增本地设置：

1. 在 `AppPreferences` 增加 key、load、save。
2. 如果涉及敏感数据，确认是否需要排除备份。
3. UI 设置入口优先放在 `ProfileScreen` 中的独立设置区块。

调整环境地址：

1. 修改 `app/build.gradle.kts` 的 `buildTypes`。
2. Debug 可以用于内网穿透 HTTP。
3. Release 必须保持 HTTPS/WSS。
4. 改完后同时跑 debug 和 release 构建。

## 14. 测试清单

每次改动至少执行：

```powershell
cd C:\DeuteriumAPP\android-app
.\gradlew.bat :app:testDebugUnitTest :app:assembleDebug :app:assembleRelease
```

账号相关手测：

- 注册验证码请求成功和失败。
- 注册成功后进入主界面。
- 登录成功、密码错误、登录态失效。
- 修改密码成功后返回登录态。

钱包相关手测：

- 余额读取成功。
- 刷新余额成功或失败。
- 最近流水不把失败显示成 0。
- 收款方搜索成功、未找到、身份无法确认。
- 转账成功、失败、余额不足、unknown 状态提示。

聊天相关手测：

- 最近 100 条加载。
- WebSocket 连接、断开、自动重连。
- WebSocket 断开后重连时补齐断线期间消息。
- 聊天页停留期间 15 秒内通过最近消息补齐静默漏收的新聊天。
- App 切走再切回聊天页后立即补齐最新消息。
- 重连提示不干扰页面。
- 发送空消息、超长消息、发送失败。
- 新消息在底部自动跟随，翻历史时显示“X 条新消息”，点击后滚到底部。
- 重复消息不会在聊天流或本地历史中重复出现。
- 服务器事件格式码被清理。
- 本地历史搜索和删除。

安装验证：

```powershell
C:\DeuteriumAPP\.tools\android-sdk\platform-tools\adb.exe devices -l
C:\DeuteriumAPP\.tools\android-sdk\platform-tools\adb.exe install -r C:\DeuteriumAPP\android-app\app\build\outputs\apk\debug\app-debug.apk
```

## 15. 发布注意事项

当前正式发布前还需要补齐：

- 正式签名配置。
- HTTPS/WSS 生产证书和 443 入口。
- Release 包安装和基础联调验证。
- 如需应用市场分发，检查明文流量、备份策略、隐私政策和网络权限说明。

不要发布 debug APK 给正式玩家。Debug APK 使用 debug 签名，并且当前连接 HTTP/80 联调环境。

## 16. 已知技术债

- `MainActivity.kt` 承担了大量 UI 代码，后续可按页面拆分到 `ui/account`、`ui/wallet`、`ui/chat`、`ui/profile`。
- Repository 目前直接持有 Compose state，适合当前小体量；后续功能增多时可引入 ViewModel。
- 本地聊天历史使用 SQLiteOpenHelper，足够轻量；如果后续需要复杂查询、分页、迁移，可再评估 Room。
- Release APK 仍未配置正式签名。
- 服务器 Pay 流水需要后端并入 `/wallet/records`，前端不能自行补全。
