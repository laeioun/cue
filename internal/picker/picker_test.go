package picker

import "testing"

func TestParseEscapeSequence(t *testing.T) {
	tests := []struct {
		name string
		seq  []byte
		want key
		ok   bool
	}{
		{name: "csi up", seq: []byte("[A"), want: keyUp, ok: true},
		{name: "csi down", seq: []byte("[B"), want: keyDown, ok: true},
		{name: "modified csi up", seq: []byte("[1;2A"), want: keyUp, ok: true},
		{name: "modified csi down", seq: []byte("[1;2B"), want: keyDown, ok: true},
		{name: "application cursor up", seq: []byte("OA"), want: keyUp, ok: true},
		{name: "application cursor down", seq: []byte("OB"), want: keyDown, ok: true},
		{name: "left", seq: []byte("[D"), want: keyLeft, ok: true},
		{name: "right", seq: []byte("[C"), want: keyRight, ok: true},
		{name: "application cursor left", seq: []byte("OD"), want: keyLeft, ok: true},
		{name: "application cursor right", seq: []byte("OC"), want: keyRight, ok: true},
		{name: "ctrl left", seq: []byte("[1;5D"), want: keyWordLeft, ok: true},
		{name: "ctrl right", seq: []byte("[1;5C"), want: keyWordRight, ok: true},
		{name: "rxvt ctrl left", seq: []byte("[5D"), want: keyWordLeft, ok: true},
		{name: "rxvt ctrl right", seq: []byte("[5C"), want: keyWordRight, ok: true},
		{name: "shift tab ignored", seq: []byte("[Z"), ok: false},
		{name: "xterm ctrl tab ignored", seq: []byte("[27;5;9~"), ok: false},
		{name: "delete", seq: []byte("[3~"), want: keyDelete, ok: true},
		{name: "ctrl delete", seq: []byte("[3;5~"), want: keyDeleteWordAt, ok: true},
		{name: "ctrl backspace csi u", seq: []byte("[127;5u"), want: keyDeleteWordBefore, ok: true},
		{name: "ctrl delete csi u", seq: []byte("[3;5u"), want: keyDeleteWordAt, ok: true},
		{name: "unknown", seq: []byte("[1;7~"), ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := parseEscapeSequence(tt.seq)
			if ok != tt.ok {
				t.Fatalf("parseEscapeSequence() ok = %v, want %v", ok, tt.ok)
			}
			if ok && got.key != tt.want {
				t.Fatalf("parseEscapeSequence() key = %v, want %v", got.key, tt.want)
			}
		})
	}
}

func TestIsCompleteEscapeSequence(t *testing.T) {
	tests := []struct {
		name string
		seq  []byte
		want bool
	}{
		{name: "partial csi", seq: []byte("["), want: false},
		{name: "complete csi", seq: []byte("[A"), want: true},
		{name: "partial delete", seq: []byte("[3"), want: false},
		{name: "complete delete", seq: []byte("[3~"), want: true},
		{name: "partial application cursor", seq: []byte("O"), want: false},
		{name: "complete application cursor", seq: []byte("OA"), want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isCompleteEscapeSequence(tt.seq); got != tt.want {
				t.Fatalf("isCompleteEscapeSequence() = %v, want %v", got, tt.want)
			}
		})
	}
}
