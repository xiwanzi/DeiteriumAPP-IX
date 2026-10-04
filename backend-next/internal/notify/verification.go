package notify

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"regexp"
)

const VerificationTemplateVersion = "identity-code-20261003-v1"
const HelpCenterURL = "https://help.deuteriumix.com"
const SupportEmail = "support@deuteriumix.com"

type VerificationMail struct {
	TemplateVersion string `json:"templateVersion"`
	Purpose         string `json:"purpose"`
	GameID          string `json:"gameId"`
	Code            string `json:"code"`
}

var verificationCodePattern = regexp.MustCompile(`^[0-9]{6}$`)
var verificationTemplate = template.Must(template.New("verification").Parse(`<!doctype html><html lang="zh-CN"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><meta name="color-scheme" content="light"><meta name="supported-color-schemes" content="light"><style>@media(max-width:600px){.mail-pad{padding:32px 24px!important}.mail-code{font-size:42px!important;letter-spacing:7px!important}}@media(prefers-color-scheme:dark){.mail-light{background:#ffffff!important;color:#20252b!important}.mail-muted{color:#68727e!important}}</style></head><body class="mail-light" bgcolor="#ffffff" style="margin:0;background-color:#ffffff!important;background-image:linear-gradient(#ffffff,#ffffff);color:#20252b;font-family:'D9 Mail Digits','HarmonyOS Sans SC','HarmonyOS Sans SC Complete','Microsoft YaHei','PingFang SC','Segoe UI',sans-serif"><table role="presentation" width="100%" cellpadding="0" cellspacing="0" bgcolor="#ffffff" style="background-color:#ffffff!important"><tr><td align="center"><table role="presentation" width="600" cellpadding="0" cellspacing="0" style="width:100%;max-width:600px"><tr><td class="mail-pad mail-light" style="padding:54px 36px 40px;background-color:#ffffff!important;color:#20252b!important">
<h1 style="margin:0 0 28px;font-size:25px;font-weight:600;line-height:1.5;color:#20252b!important;-webkit-text-fill-color:#20252b">您的 Deuterium ID 验证码：</h1>
<p class="mail-code" aria-label="验证码 {{.Code}}" style="margin:0 0 34px;font-family:Consolas,'Courier New',monospace;font-size:48px;font-weight:700;letter-spacing:9px;line-height:1.3;color:#171b20!important;-webkit-text-fill-color:#171b20">{{.Code}}</p>
<p style="margin:0 0 10px;font-size:15px;line-height:1.9;color:#20252b!important">{{.Lead}}</p>
<p style="margin:0 0 22px;font-size:18px;line-height:1.6;font-weight:600;color:#20252b!important">{{.GameID}}</p>
<p style="margin:0 0 18px;font-size:14px;line-height:1.95;color:#414a55!important">{{.Instruction}}</p>
<p style="margin:0 0 30px;font-size:14px;line-height:1.95;color:#414a55!important">验证码将在 <strong>10 分钟</strong>后失效。请勿向任何人透露验证码，包括自称 Deuterium IX 工作人员的人。</p>
<p style="margin:0;font-size:14px;line-height:1.95;color:#414a55!important">感谢您的信任与支持。<br>Deuterium IX 团队</p>
<table role="presentation" cellpadding="0" cellspacing="0" style="margin-top:38px"><tr><td width="64" style="width:64px;padding-right:0"><img src="cid:deuterium-ix-emblem-small-v1" width="48" height="48" alt="Deuterium IX 标志" style="display:block;width:48px;height:48px;border:0"></td><td style="font-family:'D9 Mail Digits','HarmonyOS Sans SC','HarmonyOS Sans SC Complete','Microsoft YaHei','PingFang SC','Segoe UI',sans-serif;font-size:19px;line-height:1.5;font-weight:700;letter-spacing:1px;color:#20252b!important"><img src="cid:deuterium-ix-wordmark-v1" width="143" height="26" alt="DEUTERIUM IX" style="display:block;width:143px;height:26px;border:0"></td></tr></table>
<p class="mail-muted" style="margin:15px 0 25px;font-size:11px;line-height:1.8;letter-spacing:.35px;color:#7c858f!important">© 2026 DEUTERIUM IX. ALL RIGHTS RESERVED.</p>
<p class="mail-muted" style="margin:0;padding-top:22px;border-top:1px solid #e7ebef;font-size:12px;line-height:1.9;color:#7c858f!important">请勿回复本邮件，此发信地址无法接收或处理您的回复。<br>如需帮助，请访问<a href="https://help.deuteriumix.com" style="color:#2874d0!important;text-decoration:underline">帮助中心</a>，或发送邮件至 <a href="mailto:support@deuteriumix.com" style="color:#2874d0!important;text-decoration:underline">support@deuteriumix.com</a>。</p>
</td></tr></table></td></tr></table></body></html>`))

func VerificationSubject(purpose string) (string, error) {
	action := map[string]string{"register": "注册账号", "password_reset": "找回密码", "password_change": "更改密码"}[purpose]
	if action == "" {
		return "", errors.New("验证码用途无效。")
	}
	return "Deuterium ID · " + action + "验证码", nil
}
func RenderVerification(in VerificationMail) (Message, error) {
	if in.TemplateVersion != VerificationTemplateVersion || !verificationCodePattern.MatchString(in.Code) || in.GameID == "" {
		return Message{}, errors.New("验证码邮件参数无效。")
	}
	subject, err := VerificationSubject(in.Purpose)
	if err != nil {
		return Message{}, err
	}
	lead, instruction := "", ""
	switch in.Purpose {
	case "register":
		lead = "您正在为以下游戏 ID 注册 Deuterium ID："
		instruction = "如果游戏 ID 填写有误，请返回注册页面修改，并重新获取验证码。若非您本人申请注册，请忽略此邮件。"
	case "password_reset":
		lead = "您正在找回以下 Deuterium ID 的密码："
		instruction = "请在找回密码页面输入验证码，并设置新密码。若非您本人操作，请忽略此邮件；您的密码不会因此改变。"
	case "password_change":
		lead = "您正在更改以下 Deuterium ID 的密码："
		instruction = "请在更改密码页面输入验证码，完成密码更新。若非您本人操作，请勿使用或转发验证码，并及时检查账号安全或联系我们。"
	}
	data := struct{ Code, GameID, Lead, Instruction string }{in.Code, in.GameID, lead, instruction}
	var html bytes.Buffer
	if err = verificationTemplate.Execute(&html, data); err != nil {
		return Message{}, err
	}
	images, err := mailBrandImages(false)
	if err != nil {
		return Message{}, err
	}
	return Message{Subject: subject, FromName: "Deuterium IX", HTML: mailHTML(html.String()), Text: fmt.Sprintf("您的 Deuterium ID 验证码：\n\n%s\n\n%s\n%s\n\n%s\n\n验证码将在 10 分钟后失效。请勿向任何人透露验证码，包括自称 Deuterium IX 工作人员的人。\n\n感谢您的信任与支持。\nDeuterium IX 团队\n\nDEUTERIUM IX\n© 2026 DEUTERIUM IX. ALL RIGHTS RESERVED.\n\n请勿回复本邮件，此发信地址无法接收或处理您的回复。\n如需帮助，请访问帮助中心：%s，或发送邮件至 %s。", in.Code, lead, in.GameID, instruction, HelpCenterURL, SupportEmail), Images: images, Headers: map[string]string{"Auto-Submitted": "auto-generated", "X-Auto-Response-Suppress": "All"}}, nil
}
func DecodeVerificationPayload(payload string) (Message, error) {
	var in VerificationMail
	if err := json.Unmarshal([]byte(payload), &in); err != nil {
		return Message{}, err
	}
	return RenderVerification(in)
}
