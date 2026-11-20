package store

import "testing"

func TestKeybindingConflictReplayAndAcknowledgements(t *testing.T) {
	row := Keybinding{Value: "key.keyboard.h", Observed: "key.keyboard.g", Revision: 9}
	old := KeyChange{Before: "key.keyboard.t", Value: "key.keyboard.g"}
	if got := mergeKey(row, old); got != row {
		t.Fatal("an old local report reopened a resolved conflict", got)
	}
	if got := mergeKey(row, KeyChange{Value: "key.keyboard.h", Ack: 8}); got != row {
		t.Fatal("outdated acknowledgement", got)
	}
	if got := mergeKey(row, KeyChange{Value: "key.keyboard.h", Ack: 9}); got.Observed != row.Value || got.Conflict {
		t.Fatal(got)
	}
}
func TestKeybindingInputTypes(t *testing.T) {
	for _, v := range []string{"key.keyboard.unknown", "key.keyboard.space", "key.keyboard.keypad.4:ALT", "key.keyboard.f25", "key.mouse.5:CONTROL", "scancode.142:SHIFT"} {
		if !ValidKeyValue(v) {
			t.Error("rejected supported binding", v)
		}
	}
	for _, v := range []string{"key.keyboard.unknown:ALT", "key.keyboard.not.a.key", "key.mouse.0", "key.keyboard.g:GUI", "key.keyboard.q:ALT:SHIFT", "scancode.-1", "options.txt"} {
		if ValidKeyValue(v) {
			t.Error("accepted unsupported binding", v)
		}
	}
}
