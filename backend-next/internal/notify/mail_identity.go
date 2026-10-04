package notify

import (
	"bytes"
	"html/template"
	"strings"
)

var mailFontCSS, _ = admissionFiles.ReadFile("templates/mail-fonts.css")
var systemNoticeTemplate = template.Must(template.ParseFS(admissionFiles, "templates/system-notice-v1.html"))

func mailHTML(html string) string {
	return strings.Replace(html, "</head>", "<style id=\"deuterium-mail-fonts\">"+string(mailFontCSS)+"</style></head>", 1)
}

// Plain administrative notices and relay messages receive the same readable
// typography. Their text alternative, recipients and threading remain intact.
func RenderSystemMessage(message Message) (Message, error) {
	if message.HTML != "" {
		return message, nil
	}
	var html bytes.Buffer
	if err := systemNoticeTemplate.ExecuteTemplate(&html, "system-notice-v1.html", message); err != nil {
		return Message{}, err
	}
	message.HTML = mailHTML(html.String())
	return message, nil
}
