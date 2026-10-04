# 验证码邮件与邮件卡片候选验收

2026-10-04，本机候选，未发布。沿用 `.worktrees/mail-identity-support` / `xiwanzi/mail-identity-support`。

## 模板

统一采用参考图的简洁邮件排版：标题“您的 Deuterium ID 验证码：”、大号六位验证码、操作对应的游戏 ID、用途与非本人操作说明、10 分钟有效期和礼貌结尾。

- 注册：提示游戏 ID 有误时返回修改，非本人申请可忽略。
- 找回密码：说明输入验证码后设置新密码，未使用验证码不会改变现有密码。
- 更改密码：对应已登录账号的密码更新，非本人操作时检查账号安全。
- 使用已有 132px 品牌徽标，以 CID 附件内嵌，配 `DEUTERIUM IX` 字标。
- 版权沿用官网维护源码页脚：`© 2026 DEUTERIUM IX. ALL RIGHTS RESERVED.`。
- 底部注明不处理回复，链接 `https://help.deuteriumix.com` 和 `support@deuteriumix.com`。用户明确只预留帮助中心链接，本轮没有创建 Help 网站或配置其域名。

真实验证码只以散列保存，模板参数（包括验证码）加密进入现有队列，发送后清除密文。新代码队列 payload 为带版本的结构化参数；原候选尚未部署，不涉及生产旧明文验证码队列迁移。

## 密码用途与界面

新增已登录的 password-change-code / password-change 路径，验证码绑定当前账号与 password_change 用途；不能交给找回密码接口或其他账号使用。成功后撤销该账号的全部会话。旧账号尚未绑定邮箱时，继续复用原 Core 游戏验证码投递能力。

App 账号设置与 Web 个人页接入独立更改密码流程。后台通知模板可分别预览三种用途。

工单使用邮件卡片，展示发件人、收件人、时间和投递状态，内部备注使用独立样式；每个工单的邮件回复草稿与内部备注草稿分别保存，不随切换通道串用。

邮件归入工单记录、公开回复与内部备注区分的依据是 [Zendesk 官方说明](https://support.zendesk.com/hc/en-us/articles/4408828489370-Adding-comments-to-tickets)。具体界面是本项目自行实现，不是 Zendesk 的成品组件或逐像素复制。

## 本次验证

- 三种模板的用途文案、六位数/前导零、品牌、帮助链接、HTML 转义、非法用途拒绝通过。
- MIME 解析确认纯文本、HTML 和内嵌 Logo 均存在。
- 密码用途隔离、跨账号拒绝、会话撤销、新旧密码登录回归通过；邮箱注册与游戏绑定相关 race 回归通过。
- Go 普通测试与 vet 通过；Web 71 项测试及构建通过。
- Android `:ui-lab:assembleDebug :ui-lab:testDebugUnitTest` 通过，152 项测试。版本仍为 2.0.14 / 21400，debug 包 SHA-256 `c8bb7dc6c9873271ebc8d7f9678c2dd58a1a4d6c3852f727c2adc0747dfaa118`，不是线上升级包。
- 浏览器验证内部备注草稿切换到邮件回复为空，切回备注仍保留原文；未发送测试草稿。
- 浏览器检查桌面与 390px 窄屏邮件模板；本机静态预览仅有不存在的 favicon 请求，不影响邮件内容和内嵌图片。

预览产物位于 `output/verification-templates/`：`verification-register.html`、`verification-password_reset.html`、`verification-password_change.html`，对应 PNG 和 `mail-cards.png`。

本轮未向真实邮箱发送测试邮件，未做手机真机验收，未推送、合并或部署模板/界面代码。账号清理是另一个已经执行的操作，见[物理清理记录](../deployment/account-purge-20261004.md)。
