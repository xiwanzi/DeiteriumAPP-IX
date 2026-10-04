# 工单收件回执样式

后续已完成真实收发接通，并按用户反馈改为 HTTPS 图片和无摘要回执；见[最终上线记录](../deployment/mail-system-live-20261004.md)。下文为此前阶段验证，不作为最新发布状态。

后续已统一为[官网字体与品牌规范](../contracts/mail-visual-standard.md)，见[最新字体验收](mail-typography-20261004.md)。下文保留早期样式和实现记录，早期字号不再作为当前规范。

2026-10-04。本机模板调整，尚未推送或部署；线上仍为 App 2.0.15 / Web 2.0.21 对应 Go。

用户最新提供 Google Play 支持邮件作为视觉参考，明确要求简单、干净、没有高饱和点缀，并去掉「待审查」。此要求覆盖上一版终末地装饰方向。

## 实现

- 白底、浅灰品牌栏和深灰文字，徽章和字标与验证码邮件统一：46px 徽章、19px / 700 字重 / 1px 字距的 DEUTERIUM IX；工单编号使用加粗等宽字体，桌面 30px、手机 28px。页面层级仅用字号和段落间距组织，无荧黄、深色外框、状态色块和侧边装饰线。
- 展示「工单已收到」与完整十二位工单编号，正文包含简短确认、问题摘要、回复方式及礼貌结尾。HTML 正文、隐藏预览摘要及纯文本均已去除「待审查」和当前状态行。
- 沿用纯邮件处理：补充信息直接回复邮件并保留主题。没有增加在线工单按钮、虚构状态查询地址或响应时限承诺。
- Logo 使用既有约 32.5 KB PNG，以 CID 随信发送。后台预览转换为内嵌图片，避免依赖远程图片加载；帮助链接及版权保留。
- 沿用审核邮件的固定配色处理、HTML 安全转义与纯文本版本；回复地址、引用头、自动回复标记及邮件主题中的工单号保持原协议。

## 验证

- `go test ./internal/notify ./internal/httpapi`、对应包 `go vet` 和 `git diff --check` 通过。
- 扩展既有工单回执测试：验证 CID 附件、图片大小、浏览器预览转换，同时保留内容转义、Reply-To / In-Reply-To / References、防头注入检查。
- 浏览器查看 760px 桌面和 390px 手机预览，徽章加载成功、完整编号可读。320 / 390 / 600 / 760px 均无横向溢出，320px 长英文标题和长连续摘要能够换行。
- 模拟深色偏好后正文背景仍为 `rgb(255, 255, 255)`；未冒充实际 QQ、Outlook、Gmail 等邮件客户端的投递或配色验收。
- 未接入真实邮箱、发送邮件或修改线上账号、工单和配置；未重建 App / Web。

## 本机预览

- `output/support-receipt/support-received.html`：正常示例。
- `output/support-receipt/desktop.png`、`mobile.png`：桌面和手机截图。
- `output/support-receipt/dark-client.png`、`long-mobile.png`：深色偏好和长内容检查。

重新生成 HTML：为通知包测试设置 `DEUTERIUM_MAIL_PREVIEW_DIR`，运行 `TestTicketReceiptEscapingReplyToAndThreadHeaders`。生成物留在忽略目录，不加入发布包。
