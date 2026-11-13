# App 2.0.30 正式包验收

2026-11-14（Asia/Singapore）。用户确认手机上的 `2.0.30-glass-preview.17`，授权发布正式版，要求不写版本说明。正式 APK 的 `releaseNotes` 使用空字符串。

## 发布内容与边界

将用户确认的玻璃控件、导航材质、聊天分组头像、聊天背景渐退与公共聊天去标题版本转为 **2.0.30（23000）**。运行代码与确认的 pre17 提交 `1c5c39e925f7f267b26dd145ac6b42eca12a5ec0` 完全相同，仅提升版本名称与版本号；正式构建提交 `96ac738467134a2d551d1645a0b0051518dac983`。

App 包名 `com.deuterium.app.uilab`，API 为 `https://47.103.99.34`，继续使用原证书和非 debuggable、启用 R8 的 performance 构件。保留上游 AndroidLiquidGlass 的 Apache-2.0 许可证。

本轮不改变 Web、Go 程序、数据库结构或游戏插件。发布仅增加 APK 和一条正式更新记录；现有 API 通过重启加载更新清单，保留原记录和配置。此文件记录内部验收，不作为客户端版本说明。

## 验证

构建目录 `android-app`，Java 17、Android SDK 35.0.0、Gradle wrapper，执行：

```text
gradlew.bat -I C:/DeuteriumAPP/output/android-liquid-glass-preview/preview-repositories.gradle :ui-lab:assemblePerformance :ui-lab:testPerformanceUnitTest :ui-lab:lintPerformance --offline --console=plain
```

- 正式包构建成功，185 项单元测试零失败、零错误；lint 无 Error/Fatal。
- `git diff 1c5c39e925f7f267b26dd145ac6b42eca12a5ec0 HEAD -- android-app/ui-lab/src/main` 为空；正式包运行内容与用户确认的预览一致。
- AAPT 核实版本、包名、minSdk 26、targetSdk 35 和非调试标记；APK v2 签名通过，与正式版 2.0.29 证书一致。
- 在全新独立 Android 15 AVD 中安装正式 2.0.29，再覆盖 2.0.30；应用 UID 与私有文件标记保留，冷启动成功、进程正常、crash buffer 无异常。没有使用真实账号，也未修改既有模拟器中的 App 数据。
- 手机上的 pre17 已由用户确认；本次独立模拟器覆盖检查不代替真机视觉结论。

APK：`C:/DeuteriumAPP/delivery/Deuterium-2.0.30/Deuterium-2.0.30-23000.apk`，9,174,398 字节。

SHA-256：`6b9bbf22d175916ddde684cb89f100b0923929fdcf7e2af128446b459e31a523`。

证书 SHA-256：`04a213066a78a968c2a1328a153a0967f99d117e0a826285e3f4af705db6f2f3`。

构建、升级和发布回执位于本机 `C:/DeuteriumAPP/output/app230-release/`；构件清单与 R8 mapping 压缩文件位于交付目录。前序分轮证据见 [第十四版](glass-preview14-calibration-20261113.md)、[第十五/十六版](chat-edges-preview16-20261113.md)。前序文档中的版本效果和“未发布”状态各自对应当时版本。
