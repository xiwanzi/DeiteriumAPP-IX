package notify

import (
	"errors"
	"net/textproto"
	"strings"
)

var ErrInvalidRecipient = errors.New("邮箱地址无效")

// Interpret only a documented DM invalid-address response. Do not persist raw
// provider replies, which may include recipient addresses or authentication data.
func smtpRecipientError(host string, err error, fallback string) error {
	var reply *textproto.Error
	if aliyunDMHost(host) && errors.As(err, &reply) {
		message := strings.ToLower(reply.Msg)
		if reply.Code == 559 && strings.Contains(message, "invalid rcptto") || reply.Code == 563 && strings.Contains(message, "account-level bounce suppression list") {
			return ErrInvalidRecipient
		}
	}
	return errors.New(fallback)
}

func aliyunDMHost(host string) bool {
	switch strings.ToLower(host) {
	case "smtpdm.aliyun.com", "smtpdm-ap-southeast-1.aliyun.com", "smtpdm-us-east-1.aliyun.com", "smtpdm-eu-central-1.aliyun.com":
		return true
	default:
		return false
	}
}
