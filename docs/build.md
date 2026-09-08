# 构建指南

本仓库保留原模块名和 Go import 路径，避免仅因归档改变发布代码。外部组件先按 `components.lock.json` 的提交检出；不要直接用默认分支最新版本替换配套依赖。

## Android

JDK 17、Android SDK Platform 36、Build Tools 35.0.0；Wrapper Gradle 8.7、AGP 8.6.1、Kotlin 2.3.10。设置本机 `ANDROID_HOME` 或忽略的 `android-app/local.properties`。首次构建需下载 Maven/Google 依赖；离线模式只适用于已具备完整缓存的机器。

```powershell
cd android-app
./gradlew.bat :ui-lab:assembleDebug :ui-lab:testDebugUnitTest :ui-lab:lintDebug --console=plain
```

APK 位于 `ui-lab/build/outputs/apk/debug/`。当前包名 `com.deuterium.app.uilab`，2.0.2 (20200)，Android 8.0+。`app` 是保留的旧客户端模块，现行发布只使用 `ui-lab`。自定义 API Origin 用 `-PdeuteriumApiBase=https://你的入口`，更新配置另见 `ui-lab/src/main/res/values/update_config.xml`。

当前线上测试 APK 使用既有 debug 签名证书。构建成功不代表可覆盖安装：新机器须从受限交付渠道取得同一签名材料，校验签名摘要、包名和递增 versionCode；不得将密钥提交到仓库。只归档/验证时不提高版本、不重新发布。Android instrumentation 夹具有联网能力，连接真实服务前按测试内容单独授权。

## 网站

使用满足 Vite 8 package-lock 约束的 Node（本次验证为 Node 24.14.1），在 `web-app` 执行：

```sh
npm ci
npm test
npm run build
```

发布目录为 `web-app/dist`；生产 Nginx 直接代理 Go，网站没有常驻 Node 服务。`server.mjs`、`gateway.mjs` 为旧开发/网关路径，实际部署见 [网页部署说明](../web-app/docs/deployment-v2.md)。浏览器 API 使用同源地址；网站包不能包含服务密钥。

## Go

Go 1.27.1（go.mod 要求 1.27.0）。在 `backend-next` 执行：

```sh
go test ./...
go vet ./...
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags='-s -w' -o bin/deuterium ./cmd/deuterium
```

数据库集成回归另用 `go test -race -tags integration ./...`。仅设置专用 `DEUTERIUM_TEST_DSN`，必须是回环地址且未指定数据库、允许建临时库。测试拒绝远端库，不用生产 `DEUTERIUM_DSN`。Windows race 另需 C 编译器；不把未执行的 integration 算作通过。

## Mail → Core → Sync

Java 21、Maven 3.9；Mail UI 使用自己的 Gradle Wrapper。独立仓库可以检出到任意目录，下例的 `external` 是忽略的本机目录：

```sh
git clone https://github.com/Deuterium-IX/Deuterium-Mail.git external/mail
git clone https://github.com/Deuterium-IX/YouerModSync-Patches.git external/sync
# 分别 git checkout components.lock.json 中的 commit
mvn -f external/mail/pom.xml clean verify install
mvn -f deuterium-core/pom.xml verify install
```

Mail 插件 API 版本 0.6.1，契约/桥/UI 0.6.0。UI 在 `external/mail/mail-ui` 执行 `gradlew.bat clean build`，保留其 vendor 中固定的 UIKit。Core 对 Mail 只依赖 provided API，不打包邮箱实现。

Sync .3 必须使用摘要为 `14fc3674b38153ec3db2c56fa8ee70a9662e854e7f8e5fdec9c962fb5dc059f4` 的 .2 输入 JAR，以及 Youer 1.21.1 开发 JAR：

```powershell
./external/sync/core-integration/build.ps1 -InputJar /secure/YouerModSync-0.7-deuterium.2.jar -YouerDevJar /secure/youerdev-1.21.1.jar -CoreProject ./deuterium-core
```

输出 `external/sync/core-integration/target/YouerModSync-0.7-deuterium.3.jar`。输入请放在独立目录，不能直接把 Maven 仓库中的同坐标 JAR 再安装回自身。仓库保留 .1→.2 修补器；更早原始插件二进制从受控交付渠道取得，不能用其他同名文件代替。Sync 9 项数据库测试需要显式 `DEUTERIUM_SYNC_TEST_URL` 与 `DEUTERIUM_SYNC_TEST_PASSWORD`，缺失时跳过。

## XConomy

本项目持久化资金扩展位于 `adapters/xconomy`，上游 XConomy 2.26.3 对应源码在 `third_party/XConomy-2.26.3`，保持 GPL-3.0-or-later。原始 JAR 摘要必须为 `0e3695f75f9d8769bb365d6c60fe48d169baf162b0bed423b456acef0a53584f`。

```sh
mvn install:install-file -Dfile=/secure/XConomy-Bukkit-2.26.3.jar -DgroupId=me.yic -DartifactId=xconomy-bukkit-input -Dversion=2.26.3 -Dpackaging=jar -DgeneratePom=true
mvn -f adapters/xconomy/pom.xml verify
python adapters/xconomy/build_patch.py --input /secure/XConomy-Bukkit-2.26.3.jar --output adapters/xconomy/target/XConomy-Bukkit-2.26.3-deuterium.1.jar
```

补丁不覆盖输入。交付源码、上游许可证和原版源码须配套；资金扩展不是从本仓库重新编译完整第三方多平台项目。详见 [XConomy 说明](../adapters/xconomy/README.md)。
