# Getting Started / 快速开始

Last updated / 更新时间：2026-05-02

## 中文

本文面向想要本地构建、阅读或二次开发 DeuteriumAPP 的开发者。公开仓库不包含生产配置、真实 token、APK、插件 jar 或本地 SDK。

### 1. 前置环境

建议环境：

- Windows PowerShell
- JDK 17
- Android Studio 或 Android SDK
- MySQL 8 或 MariaDB
- Minecraft 测试服务器：Mohist/Spigot 1.20.1
- Vault
- XConomy 或其他 Vault economy provider

项目包含 Gradle Wrapper；一般不需要全局安装 Gradle。

### 2. 克隆与阅读

```powershell
git clone https://github.com/<owner>/<repo>.git
cd <repo>
```

建议阅读顺序：

1. `README.md`
2. `docs/technical-architecture.md`
3. `docs/contracts/app-backend-api-v1.md`
4. `docs/development-handbook.md`

### 3. 后端配置

复制示例配置：

```powershell
cd backend-api
Copy-Item src\main\dist\config\application.example.conf build\application.conf
```

实际运行包会从 `config/application.conf` 读取配置。你需要准备：

- MySQL JDBC URL
- 数据库用户和密码
- `security.sessionTokenPepper`
- `security.verificationPepper`
- `pluginBridge.token`

这些值必须是你自己的环境值，不要提交到 Git。

### 4. 构建后端

```powershell
cd backend-api
.\gradlew.bat test prepareWindowsRuntime
```

运行包输出：

```text
backend-api/build/deuterium-backend-runtime/
```

在运行包目录中放置 `config/application.conf` 后启动：

```powershell
.\migrate-db.bat
.\start-backend.bat
```

默认端口：

- App API: `28657`
- Plugin bridge: `28658`

只应该对外暴露 App API。插件桥端口应保持本机访问。

### 5. 构建 Android App

Android 需要本机 SDK。若 Android Studio 未自动生成 `android-app/local.properties`，请手动创建：

```properties
sdk.dir=C:/path/to/Android/Sdk
```

构建：

```powershell
cd android-app
.\gradlew.bat :app:testDebugUnitTest :app:assembleDebug
```

Debug APK 输出：

```text
android-app/app/build/outputs/apk/debug/app-debug.apk
```

当前 release signing 未配置，release 产物不能直接作为正式发布包。

### 6. 构建 Minecraft 插件

```powershell
cd minecraft-plugin
.\gradlew.bat clean shadowJar
```

输出：

```text
minecraft-plugin/build/libs/deuterium-minecraft-plugin-0.1.0.jar
```

安装到测试服务器后，配置插件的 bridge URL 和 token。若后端与 Minecraft 在同一机器，bridge URL 通常是：

```text
ws://127.0.0.1:28658/bridge/plugin/ws
```

### 7. 开发注意事项

- 不要让 Android 直连数据库。
- 不要让 Android 直接修改服务器经济数据。
- 新增后端字段优先保持向后兼容。
- 数据库结构变更必须写 Flyway migration。
- 真实密钥、token、密码、keystore 不得提交。

## English

This guide is for developers who want to build, inspect, or extend DeuteriumAPP locally. The public repository does not include production config, real tokens, APKs, plugin jars, or local SDK tools.

### 1. Prerequisites

Recommended environment:

- Windows PowerShell
- JDK 17
- Android Studio or Android SDK
- MySQL 8 or MariaDB
- Minecraft test server: Mohist/Spigot 1.20.1
- Vault
- XConomy or another Vault economy provider

The project includes Gradle Wrapper, so a global Gradle install is usually unnecessary.

### 2. Clone And Read

```powershell
git clone https://github.com/<owner>/<repo>.git
cd <repo>
```

Recommended reading order:

1. `README.md`
2. `docs/technical-architecture.md`
3. `docs/contracts/app-backend-api-v1.md`
4. `docs/development-handbook.md`

### 3. Backend Config

Copy the example config:

```powershell
cd backend-api
Copy-Item src\main\dist\config\application.example.conf build\application.conf
```

The runtime reads `config/application.conf`. You need your own values for:

- MySQL JDBC URL
- database user and password
- `security.sessionTokenPepper`
- `security.verificationPepper`
- `pluginBridge.token`

Do not commit real environment values.

### 4. Build Backend

```powershell
cd backend-api
.\gradlew.bat test prepareWindowsRuntime
```

Runtime output:

```text
backend-api/build/deuterium-backend-runtime/
```

Place `config/application.conf` in the runtime directory, then start:

```powershell
.\migrate-db.bat
.\start-backend.bat
```

Default ports:

- App API: `28657`
- Plugin bridge: `28658`

Only expose the App API. Keep the plugin bridge port local.

### 5. Build Android App

Android requires a local SDK. If Android Studio does not create `android-app/local.properties`, create it manually:

```properties
sdk.dir=C:/path/to/Android/Sdk
```

Build:

```powershell
cd android-app
.\gradlew.bat :app:testDebugUnitTest :app:assembleDebug
```

Debug APK output:

```text
android-app/app/build/outputs/apk/debug/app-debug.apk
```

Release signing is not configured yet; release artifacts are not ready for public distribution.

### 6. Build Minecraft Plugin

```powershell
cd minecraft-plugin
.\gradlew.bat clean shadowJar
```

Output:

```text
minecraft-plugin/build/libs/deuterium-minecraft-plugin-0.1.0.jar
```

Install it into a test server, then configure bridge URL and token. If backend and Minecraft run on the same machine, the bridge URL is usually:

```text
ws://127.0.0.1:28658/bridge/plugin/ws
```

### 7. Development Notes

- Do not let Android connect directly to the database.
- Do not let Android mutate server economy data directly.
- Keep backend additions backward-compatible whenever possible.
- Database schema changes must use Flyway migrations.
- Do not commit real secrets, tokens, passwords, or keystores.
