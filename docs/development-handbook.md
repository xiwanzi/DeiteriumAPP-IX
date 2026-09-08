# DeuteriumAPP Development Handbook

Last updated: 2026-05-02.

This document is the maintainer entrypoint for future work. It records the current technology stack, module interfaces, build environment, build commands, and workflow rules so later agents do not need to rediscover them through trial and error.

中文说明：本文是后续维护工作的入口文档，记录当前技术栈、模块 interface、构建环境、构建命令和工作流程，避免后续代理重复试错。完整中英双语架构说明见 [Technical Architecture / 技术架构文档](technical-architecture.md)。

## 1. Workspace Map

```text
C:\DeuteriumAPP
  android-app/        Native Android app
  backend-api/        Public backend API and local plugin bridge server
  minecraft-plugin/  Minecraft server bridge plugin
  docs/               Product, architecture, interface, and delivery docs
  scripts/            Delivery and public export helpers
  delivery/           Private generated delivery packages, ignored by Git
  dist/               Private generated runtime packages, ignored by public export
  .tools/             Local SDK/runtime tools, ignored by Git
```

Read order for maintainers:

1. `AGENTS.md`
2. `CONTEXT.md`
3. Relevant PRD in `docs/prd/`
4. Relevant ADR in `docs/adr/`
5. Relevant plan in `docs/plans/`
6. This handbook and the module maintenance docs

## 2. Architecture At A Glance

The project has three runtime modules:

- Android App: player-facing native app. It never connects directly to the database or Minecraft server internals.
- Backend API: authentication, wallet orchestration, chat orchestration, persistence, public HTTP interface, App WebSocket, and local plugin bridge WebSocket.
- Minecraft plugin: local server adapter. It talks to the backend over the plugin bridge and uses Vault/XConomy for server economy operations.

Important seams:

- `BackendApi` and `ApiClient` are the Android network seam. UI and repositories should not know Retrofit or OkHttp details.
- `PluginBridge` is the backend seam to Minecraft. `WebSocketPluginBridge` is the current adapter.
- `AppChatHub` is the backend App WebSocket fanout module. It owns connected App sessions, foreground state, mention events, wallet events, and broadcast cleanup.
- `ChatFeedMerger` is the Android chat merge module. It owns duplicate handling and ordered catch-up merging.

Depth/locality rules:

- Keep player identity opaque on Android. The App stores and submits `playerRef`; it does not parse or display server UUIDs.
- Keep wallet authority behind backend + plugin bridge. The App never mutates game economy directly.
- Keep transport retry and error translation inside network/repository modules, not inside Compose UI.
- Keep public delivery/sanitization rules in scripts and docs, not in memory.

## 3. Android Frontend

Stack:

- Kotlin
- Jetpack Compose
- Material 3
- Retrofit 2.11
- OkHttp 4.12
- Gson 2.11
- Kotlin coroutines 1.8.1
- SQLiteOpenHelper for local chat history
- Single Activity, single Gradle app module

Version and SDK:

- Package: `com.deuterium.app`
- `versionCode`: `5`
- `versionName`: `1.0.3`
- `minSdk`: `26`
- `targetSdk`: `35`
- `compileSdk`: `36`
- `buildToolsVersion`: `35.0.0`
- Java/Kotlin target: JVM 17

Key files:

- `android-app/app/build.gradle.kts`: Android version, SDK, dependencies, injected environment URLs.
- `android-app/app/src/main/java/com/deuterium/app/MainActivity.kt`: Compose UI and app navigation.
- `android-app/app/src/main/java/com/deuterium/app/network/BackendApi.kt`: HTTP interface definitions.
- `android-app/app/src/main/java/com/deuterium/app/network/ApiClient.kt`: Retrofit/OkHttp adapter and WebSocket creation.
- `android-app/app/src/main/java/com/deuterium/app/repository/AppRepositories.kt`: account, wallet, and chat repositories.
- `android-app/app/src/main/java/com/deuterium/app/repository/ChatFeedMerger.kt`: chat dedupe and catch-up merge logic.
- `android-app/app/src/main/java/com/deuterium/app/repository/ChatHistoryStore.kt`: local SQLite chat history.

Build commands:

```powershell
cd C:\DeuteriumAPP\android-app
.\gradlew.bat :app:testDebugUnitTest :app:assembleDebug :app:assembleRelease
```

Debug APK:

```text
android-app/app/build/outputs/apk/debug/app-debug.apk
```

Release status:

- Release APK is currently unsigned.
- Do not distribute unsigned release APKs to players.
- Debug currently uses cleartext HTTP/WS for the private tunnel environment.

Local build environment:

- Windows PowerShell
- Gradle Wrapper 8.7
- Android Gradle Plugin 8.6.1
- Kotlin Android plugin 2.3.10
- Local SDK path in this private workspace: `C:\DeuteriumAPP\.tools\android-sdk`
- `android-app/local.properties` is local-only and must not be committed.

## 4. Backend API

Stack:

- Kotlin/JVM 1.9.24
- Ktor 2.3.12
- Netty engine
- kotlinx.serialization JSON
- MySQL/MariaDB
- HikariCP
- Exposed 0.53.0
- Flyway 10.17.0
- Argon2 password hashing
- Java 17

Key files:

- `backend-api/src/main/kotlin/com/deuterium/backend/Application.kt`: server startup and common Ktor plugins.
- `backend-api/src/main/kotlin/com/deuterium/backend/config/AppConfig.kt`: config loading and defaults.
- `backend-api/src/main/kotlin/com/deuterium/backend/routes/Routes.kt`: HTTP and WebSocket route installation.
- `backend-api/src/main/kotlin/com/deuterium/backend/repository/Repositories.kt`: account, wallet, player, and chat persistence modules.
- `backend-api/src/main/kotlin/com/deuterium/backend/bridge/WebSocketPluginBridge.kt`: backend-to-plugin bridge adapter.
- `backend-api/src/main/kotlin/com/deuterium/backend/chat/AppChatHub.kt`: App WebSocket fanout module.
- `backend-api/src/main/resources/db/migration/`: Flyway migrations.

Build and test:

```powershell
cd C:\DeuteriumAPP\backend-api
.\gradlew.bat test prepareWindowsRuntime
```

Runtime output:

```text
backend-api/build/deuterium-backend-runtime/
```

Main ports:

- Public App API: `0.0.0.0:28657`
- Local plugin bridge: `127.0.0.1:28658`
- Only expose the public App API port through the tunnel.

Current App WebSocket keepalive:

- `chat.websocketPingIntervalMillis=15000`
- `chat.websocketTimeoutMillis=35000`
- This is standard WebSocket ping/pong. It does not add business messages and does not hit the database.

Current app update config:

- `app.latestVersionCode=5`
- `app.latestVersionName=1.0.3`
- `GET /api/v1/app/update-check` reads config only and does not query the database.

## 5. Minecraft Plugin

Stack:

- Java 17
- Bukkit/Spigot API 1.20.1
- Target server: Mohist 1.20.1
- Vault API
- XConomy as Vault economy provider
- Java-WebSocket 1.5.7
- org.json 20240303
- Shadow plugin for relocated runtime dependencies

Key files:

- `minecraft-plugin/src/main/java/com/deuterium/plugin/DeuteriumBridgePlugin.java`: plugin lifecycle, events, chat, wallet monitoring.
- `minecraft-plugin/src/main/java/com/deuterium/plugin/BridgeClient.java`: WebSocket bridge client and message handling.
- `minecraft-plugin/src/main/resources/config.yml`: public-safe default config with `CHANGE_ME`.
- `minecraft-plugin/src/main/local-resources/config.yml`: private generated config, ignored and excluded from public export.

Build:

```powershell
cd C:\DeuteriumAPP\minecraft-plugin
.\gradlew.bat clean shadowJar
```

Output:

```text
minecraft-plugin/build/libs/deuterium-minecraft-plugin-0.1.0.jar
```

Runtime rules:

- Plugin bridge URL should normally remain `ws://127.0.0.1:28658/bridge/plugin/ws` when backend and Minecraft run on the same machine.
- Plugin bridge token must match backend config.
- The bridge port must not be exposed publicly.

## 6. Current Functional Surface

Account:

- registration code request through in-game private message
- registration with game ID, QQ, password, verification token/code
- login by game ID or QQ
- password reset through in-game verification
- logout and `/account/me`

Wallet and transfer:

- wallet balance read and refresh
- recipient search by opaque `playerRef`
- transfer creation with idempotent `clientRequestId`
- transfer status read
- wallet records read
- incremental wallet records with `afterRecordId`
- realtime wallet record event over App WebSocket

Chat:

- recent public chat history
- App-to-server public chat through WebSocket `chat.send`
- server-to-App public chat through `chat.message`
- server events through `server.event`
- presence through `presence.update`
- player directory
- followed players
- App-side mention notifications through selected `mentionedPlayerRefs`
- backend WebSocket keepalive for stale connection cleanup
- Android local chat history, search, and deletion

Profile:

- current user profile
- chat settings
- local chat history entry
- app update check

## 7. Standard Development Workflow

For feature work:

1. Read `AGENTS.md`, `CONTEXT.md`, and the relevant PRD/ADR/plan.
2. Confirm the module interface that will change.
3. Keep old public interfaces compatible unless the user explicitly approves a breaking change.
4. Add or update focused tests near the changed module.
5. Run the relevant build command before delivery.
6. Update docs when a new interface, config key, delivery rule, or runtime assumption is introduced.

For backend delivery:

```powershell
cd C:\DeuteriumAPP\backend-api
.\gradlew.bat test prepareWindowsRuntime
```

For Android delivery:

```powershell
cd C:\DeuteriumAPP\android-app
.\gradlew.bat :app:testDebugUnitTest :app:assembleDebug
```

For plugin delivery:

```powershell
cd C:\DeuteriumAPP\minecraft-plugin
.\gradlew.bat clean shadowJar
```

For full private production delivery, use `docs/production-delivery.md`. That document can contain private deployment facts and must not be published directly.

## 8. Known Deepening Opportunities

These are not mandatory refactors. They are places where a deeper module would improve leverage and locality.

1. Android UI module split
   - Files: `MainActivity.kt`
   - Problem: UI, navigation, and interaction state are concentrated in one large file.
   - Solution: split by page into `ui/account`, `ui/wallet`, `ui/chat`, and `ui/profile` while keeping repository interfaces unchanged.
   - Benefit: UI changes get better locality and screenshot/manual testing is easier.

2. Android repository state seam
   - Files: `AppRepositories.kt`
   - Problem: repositories currently own Compose state directly.
   - Solution: keep repository behaviour, but eventually introduce ViewModel-facing state holders when feature count grows.
   - Benefit: tests can cross a smaller interface and UI state bugs concentrate in one module.

3. Backend route depth
   - Files: `Routes.kt`, `Repositories.kt`
   - Problem: route handlers contain validation, orchestration, and response shaping in the same place.
   - Solution: for larger future features, move business orchestration into feature modules while keeping Ktor routes as thin adapters.
   - Benefit: route interfaces stay stable and domain behaviour becomes easier to test.

4. Public/private delivery seam
   - Files: `scripts/export-public-clean.ps1`, `docs/public-release.md`
   - Problem: private production docs and generated delivery packages contain deployment secrets.
   - Solution: never publish the private workspace directly; always publish a sanitized export.
   - Benefit: public release safety is repeatable instead of manual.

## 9. Files That Must Stay Private

Do not publish or commit:

- `delivery/`
- `dist/`
- `.tools/`
- `android-app/local.properties`
- `minecraft-plugin/src/main/local-resources/`
- any filled `config/application.conf`
- any APK, ZIP, JAR, keystore, token, password, pepper, or tunnel credential
