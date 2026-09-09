# Deuterium 2.0.5 部署记录

2026-09-09 13:26:44（UTC+8）。**App 2.0.5（20500）已发布在线更新，xiwanzi 账号已推送唯一更新提醒。** 修复代码与版本通过 [PR #5](https://github.com/xiwanzi/DeiteriumAPP-IX/pull/5) 合入 main，提交 `bcfe87e398e75dd199bc26992762ac9164301663`。

## 安装包与范围

- [下载 APK](https://47.103.99.34/downloads/Deuterium-2.0.5-20500-test.apk)，23,319,339 字节。
- SHA-256：`b4a5e27000ed979375047f411b1716356707d6d3b7d11a1355f85f44d8e6b939`。
- 包名 `com.deuterium.app.uilab`，版本 `2.0.5 (20500)`，Android 8.0+；与原测试版签名一致：`04a213066a78a968c2a1328a153a0967f99d117e0a826285e3f4af705db6f2f3`。
- 构建源提交 `f062b9ee8d81e516bce219396cbe9742b881c660` 与 squash 合并提交 `bcfe87e398e75` 的源码树完全相同。本机交付目录 `C:/DeuteriumAPP/delivery/Deuterium-2.0.5/`。
- 当前工作区 `C:/DeiteriumAPP-IX/.worktrees/release-v204`，发布文档分支 `release/v205`。

历史账单截至今天时由服务器确定截止时间，避免手机时钟超前触发格式错误；过去日期按北京时间整日查询。日期选择禁选未来，读取失败不再显示零交易汇总。订单、市场与委托的删除入口改为轻量红字，确认浮窗仅保留“取消 / 红色确认”。退款成功等共用结果浮层的图标、标题与说明居中。

网站仍为 2.0.4；Go 程序、Core、XConomy、Mail、Sync 与数据库结构没有更换。完整服务基线见 [2.0.4 部署记录](deuterium-2.0.4-live.md)。

## 验证

使用当前 Android Gradle wrapper：

```powershell
./gradlew.bat :ui-lab:assembleDebug :ui-lab:assembleDebugAndroidTest :ui-lab:testDebugUnitTest :ui-lab:lintDebug --offline --console=plain
```

- 126 项单测通过，0 失败/错误/跳过；Lint 0 错误、27 警告。
- 已发布 20401 经真实系统安装器升级 20500：安装前退出旧任务，安装后 Open 和桌面重新打开均正常。包名、大小、摘要、版本和签名均验证。
- 20500 实际运行浅深色页面回归：历史筛选、翻页、失败重试、删除取消/失败重试及结果浮层居中通过。隔离夹具不操作实际资金或真实记录。
- 公网完整下载 APK 与本机 SHA-256 一致，更新说明中文正确。20004、20101、20200、20300、20400、20401、20402 均可收到 20500；20500 和 20501 不提示重复安装或降级。
- `app.release:20500` 账号提醒仅 xiwanzi 一条，没有公共聊天广播。
- 载入新清单后 Go 健康检查 ready，程序摘要与 2.0.4 相同；Login/Amiya/Odyssey 三节点已重连，MEK 仍停止。实际当日账单及随后余额读取正常；临时只读验收会话已撤销并验证 401。

模拟器不替代用户手机的最终安装和视觉体验；本次没有把旧游戏领取/切服证据作为新验证。

## 发布与回退

先备份 `/etc/deuterium/releases.json` 到 `/var/backups/deuterium/app-2.0.5-20260909-20500/releases-before.json`，再上传全新命名 APK，核对服务器和公网摘要后原子替换更新清单。旧清单 SHA-256 为 `f9cd2c8588de631f33c23bcdfb7a0fd5b2e937b106d756cc8783b8835d11e8c4`。

既有 Go 在注册路由时读取一次更新清单，因此重启了一次 `deuterium.service` 以加载新清单；没有替换程序或重启游戏服务器。原网站目录、旧 APK 与所有业务数据保留。

如需撤回分发，恢复备份清单并重启 Go；已经安装 20500 的设备不会自动降级，后续修复须发布更高版本。无需回滚数据库或游戏插件。

证据：[构建](artifacts/v205/release-build.json)、[安装](artifacts/v205/install-proof.json)、[上线与更新检查](artifacts/v205/published.json)、[备份与上传](artifacts/v205/stage.json)、[重连与只读检查](artifacts/v205/live-readonly.json)。
