package notify

import (
	"errors"
	"fmt"
	"net/textproto"
	"strings"
	"testing"
)

func TestSMTPInvalidRecipientRequiresExplicitAliyunRejection(t *testing.T) {
	for _, tc := range []struct {
		host    string
		code    int
		message string
		invalid bool
	}{
		{"smtpdm.aliyun.com", 559, "Invalid rcptto [@sm070102] private@example.com", true},
		{"smtpdm.aliyun.com", 563, "Rcptto is on the account-level bounce suppression list[@ou060106]", true},
		{"smtpdm-ap-southeast-1.aliyun.com", 559, "Invalid rcptto", true},
		{"smtpdm.aliyun.com.evil.test", 559, "Invalid rcptto", false},
		{"smtp.example.com", 559, "Invalid rcptto", false},
		{"smtp.qiye.aliyun.com", 559, "Invalid rcptto", false},
		{"smtpdm.aliyun.com", 554, "Reject by content spam", false},
		{"smtpdm.aliyun.com", 552, "Daily sending quota exceeded", false},
		{"smtpdm.aliyun.com", 550, "Mailbox unavailable or access denied", false},
		{"smtpdm.aliyun.com", 559, "Temporary failure", false},
		{"smtpdm.aliyun.com", 450, "Invalid rcptto", false},
	} {
		t.Run(fmt.Sprintf("%s-%d-%s", tc.host, tc.code, tc.message), func(t *testing.T) {
			err := smtpRecipientError(tc.host, fmt.Errorf("wrapped: %w", &textproto.Error{Code: tc.code, Msg: tc.message}), "SMTP 拒绝接收邮箱")
			if errors.Is(err, ErrInvalidRecipient) != tc.invalid {
				t.Fatal("incorrect invalid address classification", err)
			}
			if strings.Contains(err.Error(), "private@example.com") {
				t.Fatal("provider reply leaked")
			}
		})
	}
	if errors.Is(smtpRecipientError("smtpdm.aliyun.com", errors.New("network timeout: Invalid rcptto"), "连接失败"), ErrInvalidRecipient) {
		t.Fatal("network error froze recipient")
	}
}
