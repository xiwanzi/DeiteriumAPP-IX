package store

import "testing"

func TestTitleDisplayValidation(t *testing.T) {
	for _, s := range []string{"溯行者", "#d9afd9参#c3bddc考#adcbde这#97d9e1样", "Title 123"} {
		if !ValidTitleDisplay(s) {
			t.Errorf("valid rejected: %q", s)
		}
	}
	for _, s := range []string{"", "#123456", "#ff000Z非法", "a\nb", "a\u202eb", "§c颜色", "abcdefghijklmnopqrstuvwxyz0123456789"} {
		if ValidTitleDisplay(s) {
			t.Errorf("invalid accepted: %q", s)
		}
	}
}
