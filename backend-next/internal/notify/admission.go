package notify

import (
	"bytes"
	"embed"
	"encoding/base64"
	"errors"
	"fmt"
	"html/template"
	"net/url"
	"strings"
)

const AdmissionTemplateVersion = "admission-20260913-v1"

//go:embed templates/*
var admissionFiles embed.FS

var admissionTemplates = template.Must(template.ParseFS(admissionFiles, "templates/*.html"))

type AdmissionMail struct {
	TemplateVersion, GameID, Decision, Reason, ApplicationURL string
	Preview                                                   bool
}

func RenderAdmission(in AdmissionMail) (Message, error) {
	if in.TemplateVersion != AdmissionTemplateVersion || in.GameID == "" || (in.Decision != "APPROVED" && in.Decision != "REJECTED") {
		return Message{}, errors.New("白名单邮件模板参数无效。")
	}
	u, err := url.Parse(in.ApplicationURL)
	if err != nil || u.Host == "" || u.User != nil || (u.Scheme != "https" && u.Scheme != "http") {
		return Message{}, errors.New("通行申请地址无效。")
	}
	name := "admission-approved-v1.html"
	m := Message{Subject: "Deuterium IX · 通行许可已签发", FromName: "Deuterium IX 管理组"}
	if in.Decision == "REJECTED" {
		if strings.TrimSpace(in.Reason) == "" {
			return Message{}, errors.New("拒绝邮件缺少审核原因。")
		}
		name = "admission-rejected-v1.html"
		m.Subject = "Deuterium IX · 申请暂未通过"
		m.Text = fmt.Sprintf("申请暂未通过\n\n%s，你好。\n感谢你申请加入 Deuterium IX。管理组已完成审核，你的本次申请暂未通过。\n\n正版玩家 ID：%s\n申请服务器：Deuterium IX\n审核结果：本次未通过\n\n未通过原因：\n%s\n\n你可以根据上述原因调整申请信息，再次提交。若对审核结果有疑问，可联系管理组确认。\n返回申请站：%s\n\n感谢你的理解，期待再次收到你的申请。\nDeuterium IX 管理组\n", in.GameID, in.GameID, in.Reason, in.ApplicationURL)
	} else {
		m.Text = fmt.Sprintf("通行许可已签发\n\n欢迎同行，%s。\n你的入服申请已通过。现在，你可以使用该正版账号进入 Deuterium IX。\n\n驶向星海，探索未知；搭建产线，让工业的齿轮转动起来。\n与伙伴并肩迎战强敌，也在玩家对决中一较高下。\n或是放慢脚步，建造心中的家园，结识新的朋友，体验这个世界的日常与风景。\n\n欢迎来到 Deuterium IX。属于你的故事，从这里开始。\n\n正版玩家 ID：%s\n通行区域：Deuterium IX\n许可状态：审核通过\n\n请使用申请时登记的正版账号登录。\n探索与创造之外，也请尊重彼此，让每个人都能自在地享受这里。\n\n期待在 Deuterium IX，与你相遇。\nDeuterium IX 管理组\n", in.GameID, in.GameID)
	}
	if in.Preview {
		m.Subject = "【模板测试】" + m.Subject
		m.Text = "【邮件模板测试，不代表实际审批结果】\n\n" + m.Text
	}
	var b bytes.Buffer
	if err = admissionTemplates.ExecuteTemplate(&b, name, in); err != nil {
		return Message{}, err
	}
	m.HTML = b.String()
	badge, err := admissionFiles.ReadFile("templates/deuterium-ix-emblem.png")
	if err != nil {
		return Message{}, err
	}
	m.Images = []InlineImage{{CID: "deuterium-ix-emblem", Filename: "deuterium-ix-emblem.png", ContentType: "image/png", Data: badge}}
	return m, nil
}

func AdmissionPreviewHTML(m Message) string {
	html := m.HTML
	for _, image := range m.Images {
		html = strings.ReplaceAll(html, "cid:"+image.CID, "data:"+image.ContentType+";base64,"+base64.StdEncoding.EncodeToString(image.Data))
	}
	return html
}
