# 账号体验与容量优化：本机候选验收

后续发布状态：2026-10-14 14:19 已发布 App 2.0.29 / Web 2.0.40 与 Go，见[上线记录](../deployment/account-experience-2.0.29-live.md)。以下保留发布前的本机验收/计划时间切片，不表示当前仍未上线。

2026-10-14，工作区 `.worktrees/account-experience-capacity`，分支 `xiwanzi/account-experience-capacity`，基于 `8487139`。**未推送、未合并、未部署；没有访问或更改生产数据库，没有在游戏服压测或重启游戏服。**

## 已完成

- 统一注册、登录、后台查询及改名同步的外部名字/UUID 查询池，合并相同在途查询、有限排队、取消与超时，保持过期名字复用和迟到证明保护。
- 登录来源流量与账号错误次数分开；基础设施繁忙不误记密码错误，释放预留时核对时间窗口，保留并发错误记录。
- 登录/注册统一签发；App 长期会话、Web 30 天，设备查看/单独退出/退出其他设备。游戏聊天短期令牌不混入设备退出。
- 超级管理员可审计地解除未验证占用；原账号邮箱登录和重新登记、持有人邮箱预留、认领邮箱更正与历史记录。已验证身份禁止在此解绑或换绑；原账号不合并、不移交。
- App/Web 新界面沿用现有主题和浮层；未验证账号也可管理设备。窄屏验证页的新增入口已调整为图标并允许工具区换行，避免按钮文字被挤成竖排。

产品边界见 [ADR 0011](../adr/0011-account-sessions-and-profile-conflicts.md)；接口与回退见[契约](../contracts/account-experience-capacity.md)。

## 百人突发与游戏通行隔离

环境：Windows、本机 Ryzen 5 9600X（6 核/12 线程）、Go 1.27.1、回环 MariaDB 11.8.8；数据库默认 12 个连接。官方查询为本地模拟，每次等待 20ms；密码使用真实现有 Argon2 参数与 2 个工作槽。通过真实 HTTP 发起同 IP 的 100 个不同账号冷登录，同时连续调用 50 次游戏通行检查。

| 轮次 | 成功/忙碌 | 登录 P50 / P95 / 最慢 | 通行检查 P95 / 最慢 |
| --- | --- | --- | --- |
| 初轮 | 100 / 0 | 1.296s / 2.337s / 2.435s | 2.11ms / 35.70ms |
| 加入限流窗口保护后的复测 | 100 / 0 | 1.735s / 2.751s / 2.838s | 3.67ms / 499.11ms |
| 分离服务端耗时后的诊断复测 | 100 / 0 | 1.231s / 2.232s / 2.317s | 1.51ms / 33.26ms |

第三轮通行处理器自身 P95 1.51ms、最慢 29.94ms；没有提高密码计算并发。第二轮的单次冷启动长尾保留在记录中，不以最后一轮覆盖。另有 100 个不同官方查询的独立排队测试全部通过，外部实际并发峰值为 8；相同在途查询合并、首个调用者取消、全部取消和排队超时验证通过。

这些数据证明本机负载下的排队与请求隔离，不是生产容量承诺，不代表真实公网正版服务、真实百人 Minecraft 进服或 TPS 已验收；也不是百人同 IP 同时获取注册验证码的测试。

日志：`output/account-experience/capacity-conflicts.log`、`capacity-final.log`、`capacity-handler.log`。

## 后端正确性

- Go 全量普通测试、`go vet ./...` 通过；账号/HTTP 包 `-race` 通过。
- 隔离 MariaDB 账号回归通过（HTTP 包 223.308s、integration 包 14.927s）：注册与验证码、新名字登录、迟到/并发名称证明、管理员改名、密码变更竞争、永久注销、稳定账号失败预算等。
- 新会话/身份冲突组合的 `-race -tags=integration` 回归通过（58.137s）；后续增量回归覆盖窗口边界、会话迁移、并发首登、验证码审计关联和认领邮箱更正记录。
- 永久 App 会话保留撤销能力；仅 active 且仍有效会话迁移，已过期/已撤销 token 不复活，网页为 30 天，游戏聊天期限不变。
- 设备列表不泄露 token/hash/CSRF；跨账号退出拒绝，网页 CSRF 拒绝，游戏 token 无法访问设备管理。重复退出不重复写撤销审计。
- 解除占用保留原 userId/playerRef、邮箱与密码；工单管理员无权执行，已验证目标拒绝；真正持有人仍需验证邮箱和游戏身份，原登记者不能抢回预留 UUID。
- 游戏归属验证码绑定目标账号、UUID 与专用用途，错误计数持久保留、一次性消费；重试返回同一处理记录。认领后禁止更改归属邮箱。
- 两个相同解除请求与一个非持有人注册并发执行，验证一次处理、相同结果与抢注拒绝。

日志位于 `output/account-experience/`：`auth-initial.log`、`auth-regression.log`、`session-conflict-race.log`、`final-security.log`、`claim-audit-final.log`、`claim-concurrent-final.log`、`go-race-final.log`。

## Web 与 Android

Web 的 88 项现有测试与生产构建通过。真实浏览器连接一次性本机数据库，完成：设备列表、确认退出、保持当前会话；管理员解除登记与更正邮箱；旧会话失效；原邮箱＋原密码重新登录；重新登记后回到游戏验证码页面。浅深色与 390px 移动视口检查通过，无横向溢出。预期的旧会话 401 不当作脚本错误；管理交互无 JavaScript 错误。

Android `:ui-lab:compileDebugKotlin :ui-lab:testDebugUnitTest` 通过，31 组、178 项、0 失败/跳过。另使用独立 QA 包和拦截 HTTP，在 Android 15 模拟器完成浅深色设备列表、确认/单设备退出、保留当前会话、解除后输入新 ID、保持未验证状态及返回验证页；六张最终截图已实际检查。初次截图因 ATD 关闭绘制而全黑，临时打开绘制后重新验证并恢复原设置；不把黑图算作视觉通过。

原生测试包通过 APK 元数据核对：`com.deuterium.app.accountqa`，`2.0.28 (22800)`；测试 APK 为 `com.deuterium.app.accountqa.test`。使用忽略目录中的 Gradle init 脚本覆盖 QA applicationId，未修改正式包名或发布版本。APK 位于 `android-app/ui-lab/build/outputs/apk/debug/ui-lab-debug.apk`，仅为隔离验收包，API 指向 `127.0.0.1:1` 并由 instrumentation 拦截，不能作为玩家发布包。

主要构建/运行命令：

```text
gradlew.bat -p android-app :ui-lab:compileDebugKotlin :ui-lab:testDebugUnitTest
gradlew.bat -p android-app -I output/account-experience/native-qa.init.gradle :ui-lab:assembleDebug :ui-lab:assembleDebugAndroidTest -PdeuteriumApiBase=http://127.0.0.1:1
adb -s emulator-5554 shell am instrument -w -e accountSessions true com.deuterium.app.accountqa.test/com.deuterium.app.uilab.LiveBackendInstrumentation
```

原生日志：`native-ui-final.log`（三项 PASS）、`native-apk.json`；原生截图：`output/account-experience/native/`。Web 截图：`output/playwright/accounts/`。模拟器交互不代替真机手感验收。

提交内保留[容量摘要](artifacts/account-experience/summary.json)、[原生浅色](artifacts/account-experience/devices-native-light.png)、[原生深色](artifacts/account-experience/devices-native-dark.png)及[手机网页](artifacts/account-experience/registration-web-mobile.png)，其余本机回执位于忽略目录。

## 发布边界

新增两份迁移，现有业务记录、密码和配置不在本轮进行生产处理。未发布 APK、Web 或 Go，也未向玩家发送通知/邮件。发布需使用正常 applicationId/服务地址重建构件、核对生产基线并按契约备份；旧程序回退存在新增会话列和身份预留兼容要求，不能直接换二进制或还原旧整库。
