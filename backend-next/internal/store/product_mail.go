package store

import (
	"fmt"
	"strings"
	"unicode"
)

const defaultProductMailBody = "购买的物品已按订单快照投递，请在支持领取的服务器打开邮箱。"

func validateProductMail(content CatalogObjectV2) error {
	for _, field := range []struct {
		key          string
		chars, bytes int
	}{{"mailTitle", 80, 240}, {"mailBody", 1000, 3000}} {
		value, present := content[field.key]
		if !present {
			continue
		} // Published products from earlier releases stay valid.
		if !catalogText(value, 0, field.chars) {
			return catalogInvalid()
		}
		text := value.(string)
		if len(text) > field.bytes {
			return catalogError(400, "MAIL_CONTENT_TOO_LONG", "游戏邮件文案过长，请减少文字或表情。")
		}
		for _, r := range text {
			if unicode.IsControl(r) && (r != '\n' || field.key == "mailTitle") {
				return catalogInvalid()
			}
		}
	}
	return nil
}

// Read only published content. The result is persisted in the order's existing
// mailboxPlan before any funds operation, so retries never re-read new drafts.
func productMailText(products map[string]CatalogRecordV2, quantities map[string]int64, orderNo string) (string, string, error) {
	title, body := "官方商城订单 "+orderNo, defaultProductMailBody
	keys := commerceSortedKeysV2(quantities)
	custom := false
	for _, id := range keys {
		content := products[id].Published
		if e := validateProductMail(content); e != nil {
			return "", "", e
		}
		custom = custom || strings.TrimSpace(catalogString(content, "mailTitle")) != "" || strings.TrimSpace(catalogString(content, "mailBody")) != ""
	}
	if !custom {
		return title, body, nil
	}
	parts := make([]string, 0, len(keys))
	for _, id := range keys {
		content := products[id].Published
		mailTitle, mailBody := strings.TrimSpace(catalogString(content, "mailTitle")), strings.TrimSpace(catalogString(content, "mailBody"))
		if mailBody == "" {
			mailBody = defaultProductMailBody
		}
		if len(keys) == 1 {
			if mailTitle != "" {
				title = mailTitle
			}
			body = mailBody
		} else {
			heading := fmt.Sprintf("%s × %d", catalogString(content, "title"), quantities[id])
			if mailTitle != "" {
				heading += "\n" + mailTitle
			}
			parts = append(parts, heading+"\n"+mailBody)
		}
	}
	if len(keys) > 1 {
		body = strings.Join(parts, "\n\n")
	}
	// Existing Core/Mail contract limits are UTF-8 bytes, not Unicode characters.
	if len(title) > 1024 || len(body) > 4096 {
		return "", "", catalogError(422, "MAIL_CONTENT_TOO_LONG", "这些商品合并后的游戏邮件正文过长，请分开购买。")
	}
	return title, body, nil
}
