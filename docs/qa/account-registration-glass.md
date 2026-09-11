# 注册验证码与登录浮层验收

2026-09-11，本地修正版，基于 App 2.0.11 / Go 2.0.10 配套源码。工作分支 `xiwanzi/auth-registration-glass`。本轮未推送、合并或部署；版本保持 `2.0.11 (21100)`，旧号和线上业务数据未修改。

## 修改结果

- 获取注册验证码只需游戏 ID、QQ；Android 不发送密码，后端忽略旧客户端附带的密码。提交注册仍校验 8–64 位密码，失败不会消费验证码。身份格式错误与密码错误改为具体提示。
- 登录页原本没有 `LocalOverlayBackdrop`，忘记密码浮层只能绘制半透明底色。现在记录登录页背景并交给原有 `IosSheet` / `SoftGlassSurface`，沿用全局 36dp 模糊、65% 不透明度、24 折射、30% 高光以及浮层开关。
- 同步请求模型、接口说明及示例。旧版 App 与网页仍可继续携带密码调用取码接口；新 App 需配套新后端才能无密码取码。

## 本次验证

1. Go：`go test ./...`、`go vet ./...` 通过。
2. 隔离 MariaDB：`go test -tags integration ./integration -run 'Test(GameRegistrationHTTP|RegistrationCode|VerificationCooldown)' -count=1` 通过。覆盖无密码取码、旧版空/短/正常密码兼容、ID/QQ 校验、重复获取限次、验证码不出现在 HTTP 响应、注册密码校验、服务器 UUID、浏览器 Cookie 边界、验证码并发消费和改密撤销会话。
3. Android：`gradlew.bat :ui-lab:assembleDebug :ui-lab:testDebugUnitTest :ui-lab:lintDebug` 通过。149 项 JVM 测试，0 失败/错误；lint 0 errors、24 warnings。
4. 原生界面：`assembleDebugAndroidTest` 后，在 Android 35 模拟器运行 `authRegistration` 检查，两个检查组均 PASS。使用隔离偏好与 HTTP 拦截，不向真实服务器发送验证码或注册账号。实际点击创建账号并在密码为空时获取验证码，捕获请求仅含 `gameId` / `qq`；填写验证码后提交，出现“密码需要 8–64 位”。
5. 忘记密码浮层：浅色、深色、关闭玻璃、打开键盘及返回检查通过；人工查看以下五张截图，模糊采样正常，背景文字不再直接透出。该证据来自模拟器，不代表真机手感验收。
6. 两份 OpenAPI YAML 和示例 JSON 可解析；取码必填仅 `gameId` / `qq`，注册仍必填密码；旧取码密码字段标记为可选且废弃。

测试包为 `ui-lab/build/outputs/apk/debug/ui-lab-debug.apk`（相对于 `android-app`），包名 `com.deuterium.app.uilab`、版本 `2.0.11 (21100)`、23,492,209 字节。它是本地 Debug 验证包，未作为新的线上更新发布。

## 截图

| 无密码取码 | 忘记密码（浅色） | 忘记密码（深色） |
| --- | --- | --- |
| [界面](artifacts/auth-registration-glass/registration-no-password.png) | [界面](artifacts/auth-registration-glass/reset-light.png) | [界面](artifacts/auth-registration-glass/reset-dark.png) |

[关闭玻璃](artifacts/auth-registration-glass/reset-glass-off.png) · [键盘打开](artifacts/auth-registration-glass/reset-keyboard.png)

后续发布应先更新后端，再交付新版 App；本轮没有数据库迁移或游戏插件变更。
