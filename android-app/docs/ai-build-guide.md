# Android 前端 AI 构建指南

本文档面向后续接手 DeuteriumAPP Android 前端的 AI 代理，说明当前机器上的构建环境、验证命令、版本号推进方式和 APK 交付路径。

本文只覆盖 `android-app/`。不要为了构建 Android 包去修改 `backend-api/`、`minecraft-plugin/` 或后端接口合同。

## 1. 当前环境事实

仓库路径：

```text
C:\DeuteriumAPP
```

Android 工程路径：

```text
C:\DeuteriumAPP\android-app
```

当前已验证环境：

- Shell：Windows PowerShell。
- JDK：Zulu OpenJDK 17.0.18。
- Gradle：使用工程自带 Gradle Wrapper，Gradle 8.7。
- Android Gradle Plugin：8.6.1。
- Kotlin Android 插件：2.3.10。
- 包名：`com.deuterium.app`。
- `minSdk`: 26。
- `targetSdk`: 35。
- `compileSdk`: 36。
- `buildToolsVersion`: 35.0.0。
- 当前版本：`versionCode 6` / `versionName 1.0.4`。

本机 Android SDK 路径：

```text
C:\DeuteriumAPP\.tools\android-sdk
```

当前 `android-app/local.properties` 内容：

```properties
sdk.dir=C:/DeuteriumAPP/.tools/android-sdk
```

`local.properties` 是本地机器配置，不应提交。换机器时，如果 Android Studio 没有自动生成，需要按实际 SDK 路径创建。

已存在 SDK 组件：

- `platforms/android-35`
- `platforms/android-36`
- `build-tools/35.0.0`
- `platform-tools`
- `cmdline-tools`

## 2. 构建前检查

先确认当前工程版本，不要只信旧文档：

```powershell
Select-String -Path C:\DeuteriumAPP\android-app\app\build.gradle.kts -Pattern "versionCode|versionName"
```

确认 Gradle Wrapper 可用：

```powershell
cd C:\DeuteriumAPP\android-app
.\gradlew.bat --version
```

如需确认 APK 内实际版本：

```powershell
C:\DeuteriumAPP\.tools\android-sdk\build-tools\35.0.0\aapt.exe dump badging C:\DeuteriumAPP\android-app\app\build\outputs\apk\debug\app-debug.apk | Select-String -Pattern "^package:"
```

## 3. 常用构建命令

进入 Android 工程目录：

```powershell
cd C:\DeuteriumAPP\android-app
```

完整验证：

```powershell
.\gradlew.bat :app:testDebugUnitTest :app:assembleDebug :app:assembleRelease
```

只构建可安装 debug APK：

```powershell
.\gradlew.bat :app:assembleDebug
```

输出路径：

```text
C:\DeuteriumAPP\android-app\app\build\outputs\apk\debug\app-debug.apk
```

Release 当前未配置正式签名，输出是 unsigned APK：

```text
C:\DeuteriumAPP\android-app\app\build\outputs\apk\release\app-release-unsigned.apk
```

不要把 unsigned release 当正式分发包交付给玩家。

## 4. 版本号推进规则

如果用户要“直接更新安装”，必须先确认 `versionCode` 比手机已安装版本更大。

版本号位置：

```text
android-app/app/build.gradle.kts
```

修改位置：

```kotlin
defaultConfig {
    versionCode = 6
    versionName = "1.0.4"
}
```

规则：

- 每次要交付可覆盖安装的新 debug APK，至少把 `versionCode` 加 1。
- `versionName` 同步推进为用户可读版本，例如 `1.0.4`。
- 改完版本号后重新运行 `.\gradlew.bat :app:assembleDebug`。
- 构建后用 `aapt dump badging` 确认 APK 内实际版本。

## 5. 安装验证

查看设备：

```powershell
C:\DeuteriumAPP\.tools\android-sdk\platform-tools\adb.exe devices -l
```

覆盖安装 debug APK：

```powershell
C:\DeuteriumAPP\.tools\android-sdk\platform-tools\adb.exe install -r C:\DeuteriumAPP\android-app\app\build\outputs\apk\debug\app-debug.apk
```

如果安装失败提示版本过旧，说明 `versionCode` 没有推进或手机上已安装更高版本。

如果安装失败提示签名不一致，需要先卸载旧包或确认旧包也是同一 debug 签名构建。

## 6. 当前接口环境

环境地址由 `app/build.gradle.kts` 的 `buildTypes` 注入到 `BuildConfig`。

Debug 当前使用 HTTP/80 联调地址：

```text
HTTP_BASE_URL = http://deuterium.s.odn.cc:80/api/v1/
CHAT_WS_URL   = ws://deuterium.s.odn.cc:80/api/v1/chat/ws
usesCleartextTraffic = true
```

Release 当前使用 HTTPS/WSS：

```text
HTTP_BASE_URL = https://deuterium.s.odn.cc/api/v1/
CHAT_WS_URL   = wss://deuterium.s.odn.cc/api/v1/chat/ws
usesCleartextTraffic = false
```

构建 Android 前端不需要启动后端。只有做真机功能联调时，才需要后端和 Minecraft 插件桥可用。

## 7. 测试与排障

当前前端单元测试至少覆盖聊天补齐合并逻辑：

```powershell
cd C:\DeuteriumAPP\android-app
.\gradlew.bat :app:testDebugUnitTest
```

测试报告路径：

```text
C:\DeuteriumAPP\android-app\app\build\test-results\testDebugUnitTest
```

常见问题：

- `SDK location not found`：检查 `android-app/local.properties` 的 `sdk.dir`。
- 下载依赖慢或失败：`settings.gradle.kts` 已配置 Aliyun、Google、Maven Central 和本地 maven 仓库；先重试构建。
- `compileSdk` 不存在：确认 `C:\DeuteriumAPP\.tools\android-sdk\platforms\android-36` 存在。
- 安装时版本不更新：推进 `versionCode`，重新构建，再用 `aapt dump badging` 确认。
- Release 包不能正式发布：当前 release APK 未签名，必须先补正式签名配置。

## 8. AI 操作边界

AI 代理构建 APK 时应遵守：

- 先读 `AGENTS.md` 和 `android-app/docs/maintenance.md`。
- 只为 Android 构建修改 `android-app/` 内必要文件。
- 不提交或写入生产密钥、数据库密码、插件 token、签名 keystore 密码。
- 不把 App 改成直连数据库或直连 Minecraft 插件。
- 构建前后说明实际执行的命令、版本号和 APK 绝对路径。
- 如果只构建 APK，不要顺手重构 UI 或修改后端合同。
