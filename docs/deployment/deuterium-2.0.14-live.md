# App 2.0.14 与聊天在线快照 Go 上线

2026-09-26 **20:00:27（UTC+8）**，Go 后端切换成功；**20:00:39** 开放 App **2.0.14（21400）** 更新。修复通过 [PR #48](https://github.com/xiwanzi/DeiteriumAPP-IX/pull/48) 合入 main，发布源码 `4e95ae0df69b02c201f0748bbf9317aed53eeb7f`。

[下载 App 2.0.14](https://47.103.99.34/downloads/Deuterium-2.0.14-21400-test.apk)。旧版可在 App 软件更新页面检查更新。

## 范围与源码

- 修复公共聊天在线玩家不显示：Go 提供登录后可访问的在线概览与名单，读取已有 Core 游戏在线快照；App 打开时加载，并在弹窗处于前台时每 5 秒刷新。区分无人在线、加载失败和服务器状态不可用，支持重试与长名单滚动。
- 修复键盘收起期间快速返回的回调空档：页面返回回调持续注册，以当前窗口 IME 可见性决定收键盘或退回页面，不使用固定延时。
- 在线身份按游戏 UUID 去重，断开节点移除，注销身份过滤；未注册的在线游戏玩家可提及，不展示私聊按钮。未知玩家引用继续拒绝。
- **Web 保持 2.0.19，Core / Gateway / Mail / XConomy / Sync 等游戏插件未更换，游戏服务未重启。** 保留新旧域名、SMTP、启动器及白名单配置。无数据库迁移。

基于本轮核实的最新主线 `cbc522d14ba5`，其 Android 源码与当时已发布 2.0.13 完全一致。Android 构建提交 `a24009cbf341f108d0b8fa3a93bb7cd46f38f5bd`，与合并提交的 Android 树一致：`58418009a2acbce61bc19f6e2e07b72d7f2ffbb6`。Go 从合并源码、干净工作区构建，`vcs.modified=false`，源码树 `b51d78be113f3e8e5d92b621d28bb32d108ff421`。

## 构建与产物

| 产物 | 字节数 | SHA-256 |
| --- | ---: | --- |
| App 2.0.14 / 21400 | 8,891,922 | `9d4d68e507ad7e84054d9ba3a833de6d9fdfb14504efdfb4fca088822072cb34` |
| Go Linux amd64 | 27,808,554 | `612326607c75e00d697e5653a56ef9709ffdf23221520ea945e7bc4f647fb368` |

App 包名 `com.deuterium.app.uilab`，沿用非 debuggable 的 performance 构建、R8、资源压缩和 Baseline Profile。签名证书 SHA-256 为 `04a213066a78a968c2a1328a153a0967f99d117e0a826285e3f4af705db6f2f3`，与已发布版本一致。Go 使用 1.27.1、CGO=0。

本机交付目录 `delivery/Deuterium-2.0.14/`，包含 APK、Go、mapping 和 manifest。构建命令：

```text
./gradlew.bat :ui-lab:assemblePerformance :ui-lab:testDebugUnitTest :ui-lab:lintDebug --console=plain
./gradlew.bat :ui-lab:assemblePerformanceAndroidTest -PdeuteriumTestBuildType=performance --console=plain
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -o ../output/release-v214/deuterium-linux-amd64 ./cmd/deuterium
```

## 本次发布验收

- 21400 构建、152 项 JVM 测试、lint 通过；lint 0 errors、24 条已有 warnings。
- 将正式 performance APK 覆盖安装到 Android 15 模拟器，运行单独的 performance 测试 APK。在线名单、空态、失败重试、浅深色、提及、40 人列表、IME 可见时页面返回和 0/30/100/500ms 连续返回全部通过。
- 初次运行正式压缩包测试时，测试工具直接调用的两个 AndroidX 辅助方法被 R8 优化移除，发生测试进程 `NoSuchMethodError`。随后仅修改测试代码，改用平台 Insets 与 Activity 返回入口，重新构建测试 APK并通过。发布 APK 内容和摘要未因此改变，未扩大生产 keep 规则。
- 在上一阶段相关 Go 测试基础上，本轮再次执行 `go test -race -tags integration ./internal/bridge ./internal/httpapi -run 'TestPresenceUses|TestChatPresence|TestSocialV2.*Public' -count=1`，通过。没有把历史测试冒充本次全量测试。
- 发布前真实在线名单接口返回 404。部署后短期鉴权会话读取新接口成功，`available=true`；该检查时点游戏玩家数为 0，概览与名单一致。没有伪造玩家来验证线上非空名单；非空、去重和刷新由隔离测试覆盖。
- Go 切换及 App 清单发布后，Login、Amiya、Odyssey、MEK 四个 Core 节点均恢复连接。校验实际运行进程文件摘要与发布 Go 一致，服务 active / ready。
- IP 和 `chat.deuteriumix.com` 正常 TLS 校验下均返回最新 App 21400、Web 2.0.19；两个在线接口匿名访问均为 401。
- 20004、21200、21300、21399 获得 21400；21400、21401 不重复更新或降级。原 14 个清单条目保留，公网 APK 大小与摘要一致。
- 发布前和停稳后确认无未结束的商城资金操作、运行中的启动器同步和未到期 AI 请求。数据库一致性备份通过解压校验，再切换 Go；验证新接口后更新 App 清单并正常重启 Go。
- 最终只读核对：**72 条账号记录、29 笔订单、5 个委托，原账号和业务资源 ID 全部保留**。1 条已有注销记录保留；schema、配置和 systemd 文件摘要不变，只有 App 发布清单按计划更新。启动器发布内容版本仍为 9。
- 所有临时验收会话均撤销并确认 401；没有发送业务聊天、邮件或定向更新通知，没有修改真实交易或恢复已撤销白名单资格。

模拟器使用真实返回键事件与页面返回入口，不能替代用户手机的厂商侧滑手势验收；极快的两个事件可能都被 IME 消费。用户机型的连续侧滑手感仍待实际升级后反馈。

## 备份与回退

备份目录 `/var/backups/deuterium/release-2.0.14-20260926-4e95ae0df69b`，目录 0700、敏感文件 0600；包括数据库、配置、清单、systemd 环境文件副本、发布前及停稳后的业务 ID 集合。

数据库压缩备份 13,901,951 字节，SHA-256 `39e78595262f2dddbed36c0b54997834b89bb96ec4c516236f55889983256ba4`。

| 项目 | 原值 | 当前值 |
| --- | --- | --- |
| Go | `/opt/deuterium/releases/chat-domain-a1064819ac0a` | `/opt/deuterium/releases/release-2.0.14-4e95ae0df69b` |
| Web | `/var/www/deuterium/releases/web-2.0.19-25a2ed8/dist` | 不变 |
| App 清单 SHA-256 | `725d226faf0b67d38ab97d35cc066a7f2c05c6580ed8afef30e1ea7b4f0904c3` | `6ca15ddaea15f69d5249ee458f5d4d30951a74f3479a4c1cfd4d1bd5aa42bcb0` |

启动失败处理已包含自动恢复旧软链接/旧清单，本次未触发。若撤回 App 分发，可恢复备份清单并重启当前 Go；保留已上传 APK。已安装 21400 的设备不能自动降级，后续修复应使用更高版本。必要时正常停止 Go、切回旧程序并启动；旧 Go 缺少在线名单接口，新 App 将显示不可用。**不恢复整库覆盖新业务或复活已注销账号。**

[构建清单](artifacts/v214/manifest.json) · [发布前](artifacts/v214/preflight.json) · [备份上传](artifacts/v214/stage.json) · [Go 切换](artifacts/v214/activated.json) · [App 发布](artifacts/v214/published.json) · [公网验证](artifacts/v214/public-verification.json) · [接口验证](artifacts/v214/account-verification.json) · [最终核对](artifacts/v214/final-state.json) · [实现验收](../qa/chat-online-back-fix.md)
