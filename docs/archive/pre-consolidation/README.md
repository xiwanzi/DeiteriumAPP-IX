<p align="center">
  <img src="design/app-icon/deuterium-app-icon-512.png" alt="DeuteriumAPP" width="112" height="112" />
</p>

<h1 align="center">DeuteriumAPP</h1>

<p align="center">
  Minecraft server companion app with a native Android client, Kotlin backend, and Bukkit plugin bridge.
</p>

<p align="center">
  <a href="LICENSE"><img alt="License" src="https://img.shields.io/badge/license-Apache%202.0-blue.svg" /></a>
  <img alt="Platform" src="https://img.shields.io/badge/platform-Android-3DDC84.svg" />
  <img alt="Backend" src="https://img.shields.io/badge/backend-Kotlin%20%2F%20Ktor-7F52FF.svg" />
  <img alt="Minecraft" src="https://img.shields.io/badge/minecraft-Bukkit%20%2F%20Spigot-62B47A.svg" />
</p>

## 项目概览

DeuteriumAPP 是为 Minecraft 服务器 `Deuterium VIII` 构建的原生 Android 伴侣应用。项目把玩家账号、服务器钱包、玩家转账、公共聊天、在线状态、通知和版本检查整合到一套可维护的客户端与服务端系统中。

仓库包含三个运行模块：

| 模块 | 路径 | 技术栈 | 职责 |
| --- | --- | --- | --- |
| Android App | `android-app/` | Kotlin, Jetpack Compose, Material 3 | 玩家登录、钱包、转账、聊天、通知和本地状态 |
| Backend API | `backend-api/` | Kotlin/JVM, Ktor, MySQL, Flyway | 鉴权、业务校验、持久化、WebSocket 和插件桥编排 |
| Minecraft Plugin Bridge | `minecraft-plugin/` | Java 17, Bukkit/Spigot API, Vault/XConomy | 游戏内验证码、聊天转发、服务器钱包操作和事件同步 |

## 功能特性

- 账号体系：玩家 ID 注册与登录、QQ 号登录、游戏内验证码、密码修改。
- 钱包与转账：余额刷新、流水记录、玩家搜索、幂等转账、失败状态反馈。
- 聊天互通：App 与服务器公共聊天互通、历史消息、在线人数、玩家目录、@ 提及通知。
- 实时同步：App WebSocket、钱包事件、聊天事件、断线后的增量补齐。
- 运维边界：后端配置化版本检查、Flyway 数据库迁移、公开源码导出脚本。

## 架构

```text
Android App
  HTTP + App WebSocket
Backend API
  Local plugin WebSocket
Minecraft Plugin Bridge
  Bukkit/Spigot + Vault/XConomy
Minecraft Server
```

## 快速开始

推荐环境：

- Windows PowerShell
- JDK 17
- Android Studio 或 Android SDK
- MySQL 8 或 MariaDB
- Minecraft 测试服务器：Mohist/Spigot 1.20.1
- Vault 与 XConomy，或其他 Vault economy provider

构建 Android App：

```powershell
cd android-app
.\gradlew.bat :app:testDebugUnitTest :app:assembleDebug
```

构建 Backend API：

```powershell
cd backend-api
.\gradlew.bat test prepareWindowsRuntime
```

构建 Minecraft 插件：

```powershell
cd minecraft-plugin
.\gradlew.bat clean shadowJar
```

完整的本地配置、数据库、插件安装和运行说明见 [快速开始](../../getting-started.md)。

## 仓库结构

```text
.
├── android-app/          # Native Android client
├── backend-api/          # Kotlin/Ktor backend service
├── minecraft-plugin/     # Bukkit/Spigot bridge plugin
├── docs/                 # Product, architecture, contracts, and operation docs
├── design/               # App icon and visual assets
└── scripts/              # Release and repository maintenance scripts
```
## License

DeuteriumAPP is licensed under the [Apache License 2.0](../../../LICENSE).
