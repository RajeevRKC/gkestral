package gemini

import (
	"fmt"
	"testing"
)

func TestEventType_String(t *testing.T) {
	tests := []struct {
		event EventType
		want  string
	}{
		{EventText, "text"},
		{EventToolCall, "tool_call"},
		{EventThoughtSignature, "thought"},
		{EventError, "error"},
		{EventDone, "done"},
		{EventGrounding, "grounding"},
		{EventType(99), fmt.Sprintf("unknown(%d)", 99)},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.event.String()
			if got != tt.want {
				t.Errorf("EventType(%d).String() = %q, want %q", int(tt.event), got, tt.want)
			}
		})
	}
}
