# 全部邮件字体与品牌验收

2026-10-04。本机候选，未推送、未部署。线上 App 2.0.15 / Web 2.0.21 对应 Go 未更换。

## 本轮范围

按[邮件视觉规范](../contracts/mail-visual-standard.md)统一中文 HarmonyOS Sans SC、数字 Novecento Sans Wide 和官网页脚字标。验证码、审核通知、自动回执、人工回复及原纯文本系统/管理邮件均已接入。

品牌徽章 48px；字标是官网 Geist 650 的实际字体渲染，使用 2 倍 PNG，以 143 × 26px 展示。联系说明 12px、版权 11px。审核邮件保留其原配色和业务文案；自动回执仍不显示待审查。

## 字体实际加载证据

通过浏览器 CDP `CSS.getPlatformFontsForNode` 检查实际使用的字体，未仅凭 CSS 字体名宣称成功：

| 位置 | 实际 PostScript 字体 | 自定义字体 |
| --- | --- | --- |
| 人工回复中文正文 | `HarmonyOS_Sans_SC` | true |
| 正文数字、十二位工单号 | `Novecentosanswide-Medium` | true |
| 六位验证码 | `Novecentosanswide-Bold` | true |
| 字标源文件渲染 | `Geist-Bold`，CSS 权重 650 | true |

已对官网六份代表性资源作匿名 GET：HarmonyOS Regular/Bold WOFF2、完整 Regular TTF、Novecento Medium/Bold WOFF2、Geist WOFF2。内容 SHA-256 与本机官网源文件匹配，均返回正确字体类型及 CORS `*`。[实际字体与资产摘要记录](artifacts/mail-typography/font-verification.json)。

另在独立浏览器上下文阻断外部字体请求：正文实际回退到 `MicrosoftYaHei`（isCustomFont=false），仍可阅读且无横向溢出；两个 CID 品牌图仍正常显示。这是回退验证，不标记为成功使用鸿蒙。

## 功能与版式

- 通知、HTTP API、存储三个包的普通测试与 vet 通过。
- 邮件工单、企业邮箱人工写信、Help 转交、审核邮件和旧 SMTP 过渡的专项集成 `-race` 通过，31.091 秒。主题、Reply-To、引用头、署名快照、正文、私密备注隔离和原队列流程保持。
- MIME 校验覆盖纯文本 / HTML 与两张 CID 品牌图；审核邮件测试仍满足 100 KiB 的消息预算。原徽章 32.5 KB，新增字标深色 4,249 字节、浅色 3,846 字节；没有把整套中文字体嵌进每封邮件。
- 八种预览均检查 320 / 390 / 760px，无横向溢出，所有内嵌品牌图加载成功。实际查看了验证码、审核通过、回执和人工回复的手机排版。
- 未验证真实 QQ、Gmail、Outlook 等客户端投递结果；没有开启新邮件通道、发送真实邮件或改动账号及业务数据。

## 本机预览

入口：`output/mail-typography/index.html`。各邮件有独立 HTML、`*-mobile.png` 与 `*-desktop.png`；`support-reply-fallback.png` 为字体请求被阻断后的效果。

生成方法：设置 `DEUTERIUM_MAIL_PREVIEW_DIR` 后运行通知包的 `TestAllMailUsesOfficialTypographyAndEmbeddedBrand`，浏览器等待 `document.fonts.ready` 后验证实际字体并截图。通用系统通知预览使用正常示例文字，安全转义测试另行断言。
