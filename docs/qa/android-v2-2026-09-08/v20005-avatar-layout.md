# 20005 头像尺寸与本人头像修复

2026-09-08。仅修复玩家头像组件，保留 20004 其余功能。

- 真实图片分支漏用占位头像原有的 42dp 约束；统一两分支尺寸，调用处显式 21dp/98dp 等尺寸继续生效。
- 联系人目录有意排除本人，市场却也依赖该目录读取本人头像。玩家头像现在优先读取当前账号的头像，不把本人混入联系人列表；更换头像后随现有状态更新。
- Android 15 模拟器、density 2.75，先在已安装 20004 复现：默认头像实际 970×308px，预期 116×116px。20005 使用真实解码的 640×640 PNG，通过默认42dp=116×116px、小头像21dp=58×58px、资料98dp=270×270px、无联系人条目的本人42dp=116×116px四项渲染检查，并人工查看截图。
- 检查代码仅在独立 androidTest APK；无账号、网络或业务写操作。构建使用 `gradlew.bat :ui-lab:assembleDebug :ui-lab:assembleDebugAndroidTest --offline`，未重跑无关业务测试。
- 发布 `2.0.0 (20005)`、包名 `com.deuterium.app.uilab`、原签名，24,027,169 bytes；SHA-256 `aafdb6e12d4a6c0a80f44664bd49bb801d8547c37573ebcc9bfb01280090b21d`。公网完整下载摘要一致，20004可更新、20005不提示自身更新。

截图（历史引用 `C:/DeuteriumAPP/output/avatar-layout-20005.png`，未纳入当前仓库） · [APK](https://47.103.99.34/downloads/Deuterium-2.0.0-20005-test.apk)
