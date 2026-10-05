# App 输入框可读性候选

2026-10-05，本机候选，基于 `4e51190`。用户要求先修改并查看截图；未提交、推送或发布，版本保持 2.0.17（21700）。

## 修改

- 共用 `RefinedField` 使用不透明的主题 `surfaceVariant` 底色，隔离浮层背景颜色。
- 默认增加 1dp 中性细边框；聚焦/错误使用 1.5dp 主题蓝/错误色边框。
- 标签从 13sp 次级灰色调整为 14sp、20sp 行高、中等字重及 `onSurface` 前景色。
- 保留浮层玻璃、圆角、业务流程；复用该组件的改密、退款说明等表单同步获得样式调整。本次视觉验收聚焦找回密码页。

## 本次验证

- `gradlew.bat :ui-lab:assembleDebug :ui-lab:assembleDebugAndroidTest --console=plain` 成功。
- APK 元数据核实：`com.deuterium.app.uilab`，2.0.17（21700），Debug，仅用于本机模拟器验证。
- Android 35、1080×2340 模拟器运行现有 `authRegistration` instrumentation，8 组检查全部 PASS，包括找回密码浅/深色、关闭玻璃、键盘与返回，以及登录/注册/验证码冷却回归。测试隔离偏好并拦截 HTTP，未发送真实验证码。
- 人工查看本次浅色、深色、聚焦与键盘截图：空输入区域轮廓可辨，标签清晰，输入底色不再跟随玻璃透色。未完成真机验收及所有复用页面的逐页视觉检查。
- `git diff --check` 通过。

## 本次模拟器截图

- [浅色](artifacts/input-readability-20261005/reset-light.png)
- [深色](artifacts/input-readability-20261005/reset-dark.png)
- [聚焦与键盘](artifacts/input-readability-20261005/reset-keyboard.png)
- [关闭玻璃](artifacts/input-readability-20261005/reset-glass-off.png)
