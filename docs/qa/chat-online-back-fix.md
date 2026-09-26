# 公共聊天在线名单与连续返回：本机修复验收

日期：2026-09-26。维护工作区 `C:/DeiteriumAPP-IX/.worktrees/chat-domain`，本轮复用已完成且干净的工作区，从最新主线建立分支 `xiwanzi/chat-online-back-fix`。本轮未推送、合并或部署。

## 最新版本依据

- `git fetch origin` 后主线为 `cbc522d14ba565db8068cff97b2b31daa8e00e91`。
- 主线 Android 树为 `4fbc9624866b84d89bcce9d7bfd5be2e46d455fc`，与已发布 App 2.0.13（21300）的 `15690c138288` Android 树逐字节一致。
- 本轮通过正常 HTTPS 只读查询 `/api/v1/app/update-check?versionCode=21200&resourceVersion=0&packageName=com.deuterium.app.uilab&channel=stable`，返回 `latestVersionCode=21300`、`latestVersionName=2.0.13`，线上 APK SHA-256 仍为 `2d063c93d865c374be426e9aa263658869c96a6aeb8f4c4de029f990f5fd398b`。
- 没有修改根目录早期源码，也没有回退其他已合入功能。候选版本暂保持 21300，不作为新的线上版本发布。

## 原因与修复

### 在线玩家

旧弹窗只显示 `Players.filter { it.online }`。它属于联系人/历史消息缓存，不是当前在线快照；Go `PublicProfileV2` 未填在线状态，且原 App 调用的 `/chat/presence` 路由不存在。点击按钮没有请求在线名单，失败也没有空态或错误说明。

补齐既有契约的 `/chat/presence` 和 `/chat/online-players`，从已鉴权、启用聊天的 Core 节点快照读取，按 UUID 去重，断连立即移除。只读解析已有 playerRef，过滤注销身份；没有快照与确认零人在线分别处理。返回不包含 UUID、QQ 等非展示必需字段。沿用 WebSocket 心跳断连机制，不新增数据库迁移或插件协议。

App 改用独立在线名单，每次打开读取，在前台显示期间每 5 秒刷新；补齐加载、空态、失败重试与长列表滚动。包含在线本人；未注册玩家隐藏私聊，保留提及。公共消息的提及校验补认已有 Core 目录中的玩家，防止首次在线但从未发言的玩家被提及时发送失败，未知 ref 仍拒绝。

### 连续返回

旧回调以 `!keyboard` 作为启用条件，而 `keyboard` 来自 IME 动画高度。输入法开始关闭后，高度仍可能大于零，页面回调因此继续禁用；第二次系统返回如果已不由输入法消费，就可能落到 Activity 的默认退出逻辑。等动画完成后回调重新启用，符合用户描述的快慢差异。

页面有返回目标时持续注册回调；收到返回时读取窗口当前 IME 可见性，可见则隐藏，不可见则回到上一页。沿用弹窗自己的返回处理，不设置固定防抖时间，也不拦截根商城页正常退出。

此处依据代码明确定位到时序漏洞，并完成模拟器返回事件回归；没有用户手机的原始崩溃日志或厂商侧滑实测，不能将其写成所有设备均复现并验收。

## 本轮验证

Windows JDK 17，SDK `C:/DeuteriumAPP/.tools/android-sdk`，项目 wrapper。

- `:ui-lab:compileDebugKotlin :ui-lab:testDebugUnitTest :ui-lab:lintDebug`：通过，152 项 JVM 测试零失败/错误；lint 0 errors、24 条已有 warnings。
- `:ui-lab:assembleDebug :ui-lab:assembleDebugAndroidTest :ui-lab:assemblePerformance`：通过。测试代码调整后重新构建并执行独立测试 APK。
- Android 15 模拟器，隔离测试会话、拦截 HTTP；`am instrument -w -e chatNavigation true com.deuterium.app.uilab.test/com.deuterium.app.uilab.LiveBackendInstrumentation` 返回 `INSTRUMENTATION_CODE: -1`：
  - 网络名单覆盖旧联系人缓存，包含本人及未注册玩家；空名单、503、快照不可用和恢复正常均通过。
  - 弹窗浅/深色、空态、失败重试、提及、40 人列表滚动至最后一人通过。
  - 0/30/100/500ms 连续返回不结束 Activity；IME 显示期间页面返回回调始终存在，IME 隐藏后能返回信息页。
  - 极快事件可能均被系统 IME 消费；测试在这种情况下补发页面返回验证，不声称每种输入法都在第二次事件导航。
  - 以上为返回键事件与 dispatcher 的检查，不等同于厂商侧滑手势实测。
- `go test ./internal/bridge ./internal/httpapi ./internal/store`：通过。
- `go test -tags integration ./internal/httpapi -run 'TestChatPresence|TestSocialV2.*Public' -count=1 -v`：3 项通过，覆盖鉴权、缺快照/空快照、注册/未注册/本人、UUID 去重、注销过滤、禁用聊天节点、断连、历史目录不能复活在线状态、首次在线玩家提及、未知 ref 拒绝及既有公共聊天提及通知。
- `go vet ./internal/bridge ./internal/httpapi ./internal/store`：通过。集成测试只使用 loopback MariaDB 随机测试库，没有操作线上业务数据。

模拟器最初使用 ATD 默认禁用绘制的配置，截图不可用于视觉验收；开启该模拟器的 `debug.hwui.drawing_enabled` 后重新执行测试并截图。证据保存在本工作区 `output/chat-online-back/`，属于本机产物。

## 构建产物与后续发布边界

`android-app/ui-lab/build/outputs/apk/performance/ui-lab-performance.apk`，8,891,922 字节，包名 `com.deuterium.app.uilab`，版本 2.0.13 / 21300。SHA-256：`e769092e1fa49995fbcfdbbaff9262c1d330d6738fee18436029dcf11247387d`。

非 debuggable 的 performance 构建，R8/资源压缩沿用当前版本；签名验证通过，证书 SHA-256 `04a213066a78a968c2a1328a153a0967f99d117e0a826285e3f4af705db6f2f3`，与现行线上包相同。它是本机验证候选，版本未递增，未加入更新清单。

上线需要配套 Go 与 App 一起发布；Web、游戏插件不需要更换。发布前另行递增 App 版本并生成最终包，在用户手机上复核公共聊天/私聊快速侧滑、先关输入法后返回、弹窗返回以及商城根页退出。没有执行线上更新、账号操作、消息发送或通知推送。
