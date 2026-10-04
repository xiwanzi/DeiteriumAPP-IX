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

func mailBrandImages(lightWordmark bool) ([]InlineImage, error) {
	wordmark := "deuterium-ix-wordmark-dark.png"
	if lightWordmark {
		wordmark = "deuterium-ix-wordmark-light.png"
	}
	images := make([]InlineImage, 0, 2)
	for _, asset := range []struct{ file, cid string }{{"deuterium-ix-emblem.png", "deuterium-ix-emblem-small-v1"}, {wordmark, "deuterium-ix-wordmark-v1"}} {
		data, err := admissionFiles.ReadFile("templates/" + asset.file)
		if err != nil {
			return nil, err
		}
		images = append(images, InlineImage{CID: asset.cid, Filename: asset.file, ContentType: "image/png", Data: data})
	}
	return images, nil
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
	images, err := mailBrandImages(false)
	if err != nil {
		return Message{}, err
	}
	message.HTML = mailHTML(html.String())
	message.Images = images
	return message, nil
}
