# 生产交付规范

本文档记录 DeuteriumAPP 当前已验证可用的生产交付方式。后续后端、Minecraft 桥接插件和 Android Debug 包交付，默认按本文档执行。

本文档的目标是避免再次出现以下问题：

- 后端目录多套一层，导致脚本路径不一致。
- 交付包没有携带 `backend/jre`，目标机器找不到 Java。
- 后端配置没有生成 `config/application.conf`。
- 后端和插件 bridge token 不一致。
- 数据库迁移失败时缺少可判断的错误信息。

## 1. 已确认生产环境

当前生产环境按旧生产包 `delivery/DeuteriumAPP-production.zip` 对齐。

- 操作系统：Windows。
- Java：交付包内置 `backend/jre/bin/java.exe`，不要依赖系统 PATH。
- MySQL：`127.0.0.1:3306`。
- 数据库名：`deuterium_app`。
- 数据库用户：`root`。
- 数据库密码：`123456`。
- App API 端口：`28657`。
- Minecraft 插件桥端口：`28658`，只监听 `127.0.0.1`。
- 公网入口只暴露 App API，不暴露插件桥。

生产 App 地址：

```text
https://deuterium.s.odn.cc/api/v1
```

生产 App WebSocket：

```text
wss://deuterium.s.odn.cc/api/v1/chat/ws
```

## 2. 交付包结构

生产交付包必须使用以下结构，不要多套 `deuterium-backend-runtime/` 目录。

```text
DeuteriumAPP-production-YYYYMMDD/
  README.md
  backend/
    migrate-db.bat
    start-backend.bat
    reset-database.bat
    config/
      application.conf
      application.example.conf
    jre/
      bin/java.exe
    lib/
      deuterium-backend-api-0.1.0.jar
      ...
    bin/
      deuterium-backend-api.bat
      deuterium-backend-api
  minecraft-plugin/
    deuterium-minecraft-plugin-0.1.0.jar
```

压缩包命名：

```text
delivery/DeuteriumAPP-production-YYYYMMDD.zip
```

示例：

```text
delivery/DeuteriumAPP-production-20260501.zip
```

## 3. 后端配置

交付包必须直接包含可用的：

```text
backend/config/application.conf
```

当前生产配置必须包含：

```properties
public.host=0.0.0.0
public.port=28657

bridge.host=127.0.0.1
bridge.port=28658

database.jdbcUrl=jdbc:mysql://127.0.0.1:3306/deuterium_app?createDatabaseIfNotExist=true&useSSL=false&serverTimezone=UTC&allowPublicKeyRetrieval=true&characterEncoding=utf8&useUnicode=true
database.user=root
database.password=<redacted>
database.maximumPoolSize=10

security.sessionTokenPepper=<redacted>
security.verificationPepper=<redacted>
security.sessionDays=30

pluginBridge.token=<redacted>
pluginBridge.requestTimeoutMillis=10000
pluginBridge.heartbeatIntervalMillis=30000
pluginBridge.staleAfterMillis=90000

chat.historyRetentionDays=30
chat.websocketPingIntervalMillis=15000
chat.websocketTimeoutMillis=35000

app.latestVersionCode=5
app.latestVersionName=1.0.3

log.level=INFO
```

注意：

- 本项目当前生产交付允许把上述配置写入 `delivery/` 交付包。
- 不要把生产配置提交到公开仓库。
- 新交付包必须沿用旧生产包中的 `sessionTokenPepper`、`verificationPepper` 和 `pluginBridge.token`，否则会影响既有 session、验证码上下文或插件连接。
- JDBC URL 必须带 `createDatabaseIfNotExist=true`，并且后端代码也会在迁移前尝试创建数据库。
- `app.latestVersionCode` 和 `app.latestVersionName` 用于 `GET /api/v1/app/update-check`，每次交付新 APK 时必须同步推进；该接口只读配置，不访问数据库。

## 4. 后端构建要求

构建前必须跑：

```powershell
cd C:\DeuteriumAPP\backend-api
.\gradlew.bat test prepareWindowsRuntime
```

构建产物来源：

```text
backend-api/build/deuterium-backend-runtime/
```

打包时复制该目录内容到交付包的：

```text
DeuteriumAPP-production-YYYYMMDD/backend/
```

然后从旧生产包或已确认 JRE 来源复制：

```text
backend/jre/
```

当前已验证 JRE 来源：

```text
delivery/_prod_ref/DeuteriumAPP-production/backend/jre/
```

## 5. Minecraft 插件构建要求

插件构建：

```powershell
cd C:\DeuteriumAPP\minecraft-plugin
.\gradlew.bat clean shadowJar
```

交付 jar 来源：

```text
minecraft-plugin/build/libs/deuterium-minecraft-plugin-0.1.0.jar
```

复制到交付包：

```text
DeuteriumAPP-production-YYYYMMDD/minecraft-plugin/deuterium-minecraft-plugin-0.1.0.jar
```

插件 jar 必须包含和后端一致的 bridge token。当前插件会优先读取 jar 内置 `config.yml` 的 bridge 地址和 token；这能避免服务器已有旧 `plugins/DeuteriumBridge/config.yml` 时使用旧 token。

当前插件内置配置应为：

```yaml
backend:
  ws-url: "ws://127.0.0.1:28658/bridge/plugin/ws"
  token: "<redacted>"
  reconnect-seconds: 5
```

## 6. 推荐打包步骤

如果没有自动脚本，按以下手动步骤生成生产包：

1. 解压或保留旧生产包到：

```text
delivery/_prod_ref/DeuteriumAPP-production/
```

2. 构建后端：

```powershell
cd C:\DeuteriumAPP\backend-api
.\gradlew.bat test prepareWindowsRuntime
```

3. 构建插件：

```powershell
cd C:\DeuteriumAPP\minecraft-plugin
.\gradlew.bat clean shadowJar
```

4. 创建新目录：

```text
delivery/DeuteriumAPP-production-YYYYMMDD/
```

5. 复制：

```text
backend-api/build/deuterium-backend-runtime/* -> delivery/DeuteriumAPP-production-YYYYMMDD/backend/
delivery/_prod_ref/DeuteriumAPP-production/backend/jre -> delivery/DeuteriumAPP-production-YYYYMMDD/backend/jre
minecraft-plugin/build/libs/deuterium-minecraft-plugin-0.1.0.jar -> delivery/DeuteriumAPP-production-YYYYMMDD/minecraft-plugin/
```

6. 写入 `backend/config/application.conf`，内容按第 3 节。

7. 写入交付包 `README.md`，说明部署顺序和端口。

8. 压缩：

```powershell
Compress-Archive -Path C:\DeuteriumAPP\delivery\DeuteriumAPP-production-YYYYMMDD -DestinationPath C:\DeuteriumAPP\delivery\DeuteriumAPP-production-YYYYMMDD.zip -CompressionLevel Optimal
```

## 7. 部署步骤

在 Windows 服务器上：

1. 解压 `DeuteriumAPP-production-YYYYMMDD.zip`。
2. 进入解压后的根目录。
3. 运行：

```bat
backend\migrate-db.bat
backend\start-backend.bat
```

4. 保持 `start-backend.bat` 窗口打开。
5. 将：

```text
minecraft-plugin\deuterium-minecraft-plugin-0.1.0.jar
```

复制到 Minecraft 服务器 `plugins/` 目录。

6. 重启 Minecraft 服务器。

## 8. 成功日志判定

后端启动成功应看到：

```text
Database: jdbc:mysql://127.0.0.1:3306/deuterium_app (MySQL 8.0)
Successfully validated 4 migrations
Schema `deuterium_app` is up to date. No migration necessary.
HikariPool-1 - Start completed.
Responding at http://127.0.0.1:28657
Responding at http://127.0.0.1:28658
```

插件桥连接成功应看到：

```text
Plugin bridge connected
```

当前已验证成功日志时间为 `2026-04-30 23:28:07`，关键信号是：

```text
Current version of schema `deuterium_app`: 4
Plugin bridge connected
```

## 9. 常见失败与处理

### 缺少 application.conf

错误：

```text
Missing backend config: ...\backend\config\application.conf
```

原因：

- 交付包没有按规范生成 `application.conf`。
- 错误使用了只含模板的开发运行包。

处理：

- 使用本文档规范重新打包。
- 不要交付只含 `application.example.conf` 的包。

### 找不到 mysql.exe

如果脚本提示 `mysql.exe was not found in PATH`，不要依赖该方式建库。

当前规范要求：

- JDBC URL 带 `createDatabaseIfNotExist=true`。
- 后端在迁移前通过 JDBC 自动执行 `CREATE DATABASE IF NOT EXISTS deuterium_app`。

所以生产包不需要 `mysql.exe` 在 PATH。

### 迁移失败

常见原因：

- MySQL 服务未启动。
- MySQL 不在 `127.0.0.1:3306`。
- 生产数据库用户名或密码不正确。
- 端口被防火墙或配置拦截。
- 数据库已有手工改坏的表结构。

新后端启动失败时会打印完整 Java 异常。排障时必须复制异常全文，而不是只复制 `Database migration failed`。

### 插件桥未连接

检查：

- 后端是否已启动并监听 `127.0.0.1:28658`。
- Minecraft 插件 jar 是否来自同一个交付包。
- 插件是否真的替换了服务器 `plugins/` 里的旧 jar。
- bridge token 是否为：

```text
<redacted>
```

### 端口占用

后端需要：

- `28657`：App API。
- `28658`：插件桥。

如果端口被占用，停止旧后端窗口或占用进程后再启动。

## 10. Android Debug APK 交付

Android 版本推进规则：

- 每次给用户新的 Debug APK，必须推进 `versionCode`。
- 小功能更新默认推进 `versionName` patch，例如 `0.1.1 -> 0.1.2`。

当前已推进版本：

```text
versionCode=5
versionName=1.0.3
```

对应后端配置：

```properties
app.latestVersionCode=5
app.latestVersionName=1.0.3
```

构建：

```powershell
cd C:\DeuteriumAPP\android-app
.\gradlew.bat :app:assembleDebug
```

产物：

```text
android-app/app/build/outputs/apk/debug/app-debug.apk
```

交付前确认：

```text
android-app/app/build/outputs/apk/debug/output-metadata.json
```

其中应包含正确的 `versionCode` 和 `versionName`。

## 11. Wiki.js OIDC 增量交付

OIDC 后端能力默认关闭。首次生产启用前必须先确认 auth origin 可访问，并且 Wiki.js callback URL 已确定。公网密码登录必须优先使用 HTTPS；内部 HTTP-only 环境必须显式启用 `oidc.allowInsecureHttp=true`。

构建启用 OIDC 的交付包示例：

```powershell
.\scripts\build-production-delivery.ps1 `
  -DbUser <PROD_DB_USER> `
  -DbPassword "<PROD_DB_PASSWORD>" `
  -EnableOidc `
  -OidcIssuer "https://auth.deuterium.cafe" `
  -OidcClientId "wikijs" `
  -OidcClientSecret "<WIKIJS_CLIENT_SECRET>" `
  -OidcRedirectUri "<WIKIJS_CALLBACK_URL>"
```

内部 HTTP-only 环境示例：

```powershell
.\scripts\build-production-delivery.ps1 `
  -DbUser <PROD_DB_USER> `
  -DbPassword "<PROD_DB_PASSWORD>" `
  -EnableOidc `
  -AllowOidcInsecureHttp `
  -OidcIssuer "http://authdeuterium.s.odn.cc" `
  -OidcClientId "wikijs" `
  -OidcClientSecret "<WIKIJS_CLIENT_SECRET>" `
  -OidcRedirectUri "https://wiki.deuterium.cafe/login/oidc/callback"
```

可回滚要求：

- 不覆盖旧后端目录，先解压到新目录。
- 沿用旧 `security.sessionTokenPepper`、`security.verificationPepper`、`pluginBridge.token`。
- 沿用 `backend/config/oidc-signing-key.json`；脚本会在本地 delivery 目录存在该文件时保留它。
- OIDC migration 只新增表，不修改旧 APP/插件表；旧后端回滚后可忽略这些表。
- 如果 auth origin 未就绪，保持 `oidc.enabled=false`，只部署代码能力，不开放密码登录入口。

Wiki.js OIDC Provider 配置见 `docs/contracts/oidc-provider-v1.md`。

## 12. 交付前检查清单

后端：

- `backend-api` 已执行 `test prepareWindowsRuntime`。
- 如为 backend-only 性能修复包，允许不重新构建或替换 `minecraft-plugin/`，但必须确认后端配置中的 `pluginBridge.token` 沿用现有生产 token。
- 交付包结构为 `backend/...`，没有多套一层运行目录。
- `backend/jre/bin/java.exe` 存在。
- `backend/config/application.conf` 存在。
- `application.conf` 数据库连接信息为当前生产环境有效值。
- `application.conf` bridge token 与插件一致。
- `application.conf` 的 `app.latestVersionCode/latestVersionName` 与本次 APK 版本一致。
- 如启用 OIDC，`oidc.issuer` 为可被浏览器和 Wiki.js 访问的 origin；公网为 HTTPS，内部 HTTP 必须同时设置 `oidc.allowInsecureHttp=true`。
- 如启用 OIDC，`oidc.redirectUri` 与 Wiki.js callback 精确一致。
- 如启用 OIDC，`backend/config/oidc-signing-key.json` 已保留或由新服务首次生成后备份。
- `backend/lib/deuterium-backend-api-0.1.0.jar` 是最新构建产物。

插件：

- 已执行 `minecraft-plugin clean shadowJar`。
- `minecraft-plugin/deuterium-minecraft-plugin-0.1.0.jar` 是最新构建产物。
- 插件内置 bridge 地址为 `ws://127.0.0.1:28658/bridge/plugin/ws`。
- 插件内置 token 与后端一致。

部署验证：

- `backend/migrate-db.bat` 成功。
- `backend/start-backend.bat` 输出 `Responding at http://127.0.0.1:28657`。
- `backend/start-backend.bat` 输出 `Responding at http://127.0.0.1:28658`。
- 启动 Minecraft 后，后端输出 `Plugin bridge connected`。
- 如启用 OIDC，`<oidc.issuer>/.well-known/openid-configuration` 返回配置中的 issuer。
- 如启用 OIDC，Wiki.js 测试账号能完成登录，且旧 Android APP 仍能登录 `/api/v1/account/me`。

Android：

- 如交付 APK，已推进版本。
- 如交付 APK，后端 `app.latestVersionCode/latestVersionName` 已同步推进。
- `:app:assembleDebug` 成功。
- APK 路径已明确给出。
