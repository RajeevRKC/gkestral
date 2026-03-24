package gemini

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestBuildSystemInstruction(t *testing.T) {
	prompt := SystemPrompt{
		Role:         "You are a Go expert.",
		Constraints:  []string{"Be concise", "No speculation"},
		OutputFormat: "Use code blocks for Go code.",
		ToolGuidance: "Read files before editing.",
	}

	msg := BuildSystemInstruction(prompt)

	if msg == nil {
		t.Fatal("expected non-nil message")
	}
	if msg.Role != "user" {
		t.Errorf("role = %q, want %q", msg.Role, "user")
	}
	if len(msg.Parts) != 1 {
		t.Fatalf("parts = %d, want 1", len(msg.Parts))
	}

	text := msg.Parts[0].Text
	if !strings.Contains(text, "Go expert") {
		t.Error("missing role in output")
	}
	if !strings.Contains(text, "Be concise") {
		t.Error("missing constraint in output")
	}
	if !strings.Contains(text, "code blocks") {
		t.Error("missing output format in output")
	}
	if !strings.Contains(text, "Read files") {
		t.Error("missing tool guidance in output")
	}
}

func TestBuildSystemInstruction_EmptyPrompt(t *testing.T) {
	msg := BuildSystemInstruction(SystemPrompt{})
	if msg == nil {
		t.Fatal("expected non-nil message")
	}
	if msg.Parts[0].Text != "You are a helpful assistant." {
		t.Errorf("empty prompt text = %q, want default", msg.Parts[0].Text)
	}
}

func TestBuildSystemInstruction_CustomOnly(t *testing.T) {
	prompt := SystemPrompt{
		CustomInstructions: "You are Gkestral, a Gemini-native assistant.",
	}
	msg := BuildSystemInstruction(prompt)
	if !strings.Contains(msg.Parts[0].Text, "Gkestral") {
		t.Error("missing custom instructions")
	}
}

func TestBuildSystemInstructionFromText(t *testing.T) {
	msg := BuildSystemInstructionFromText("Custom system prompt.")
	if msg.Role != "user" {
		t.Errorf("role = %q, want %q", msg.Role, "user")
	}
	if msg.Parts[0].Text != "Custom system prompt." {
		t.Errorf("text = %q, want %q", msg.Parts[0].Text, "Custom system prompt.")
	}
}

func TestBuildSystemInstructionFromText_Empty(t *testing.T) {
	msg := BuildSystemInstructionFromText("")
	if msg.Parts[0].Text != "You are a helpful assistant." {
		t.Errorf("empty text = %q, want default", msg.Parts[0].Text)
	}
}

func TestBuildSystemInstruction_JSONSerialization(t *testing.T) {
	msg := BuildSystemInstruction(DefaultSystemPrompt())
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	// Should serialize as a proper Message object.
	var decoded Message
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if decoded.Role != "user" {
		t.Errorf("decoded role = %q, want %q", decoded.Role, "user")
	}
	if len(decoded.Parts) != 1 {
		t.Errorf("decoded parts = %d, want 1", len(decoded.Parts))
	}
}

func TestApplySystemInstruction(t *testing.T) {
	req := &GenerateContentRequest{
		Contents: []Message{{Role: "user", Parts: []Part{{Text: "hello"}}}},
	}

	instruction := BuildSystemInstructionFromText("Be helpful.")
	ApplySystemInstruction(req, instruction)

	if req.SystemInstruction == nil {
		t.Fatal("SystemInstruction should be set")
	}
	if req.SystemInstruction.Parts[0].Text != "Be helpful." {
		t.Errorf("text = %q, want %q", req.SystemInstruction.Parts[0].Text, "Be helpful.")
	}
}

func TestDefaultSystemPrompt(t *testing.T) {
	prompt := DefaultSystemPrompt()

	if prompt.Role == "" {
		t.Error("default role should not be empty")
	}
	if len(prompt.Constraints) == 0 {
		t.Error("default should have constraints")
	}
	if prompt.OutputFormat == "" {
		t.Error("default should have output format")
	}
	if prompt.ToolGuidance == "" {
		t.Error("default should have tool guidance")
	}
}

func TestEnforceTemperature(t *testing.T) {
	tests := []struct {
		name      string
		modelID   string
		requested float64
		want      float64
	}{
		{
			name:      "3.1 Pro always 1.0",
			modelID:   "gemini-3.1-pro-preview",
			requested: 0.5,
			want:      1.0,
		},
		{
			name:      "3.1 Flash always 1.0",
			modelID:   "gemini-3.1-flash",
			requested: 0.7,
			want:      1.0,
		},
		{
			name:      "3.1 Flash Image always 1.0",
			modelID:   "gemini-3.1-flash-image-preview",
			requested: 0.3,
			want:      1.0,
		},
		{
			name:      "2.5 Flash respects requested",
			modelID:   "gemini-2.5-flash",
			requested: 0.7,
			want:      0.7,
		},
		{
			name:      "2.5 Pro respects requested",
			modelID:   "gemini-2.5-pro",
			requested: 0.3,
			want:      0.3,
		},
		{
			name:      "2.x clamp negative to 0",
			modelID:   "gemini-2.5-flash",
			requested: -0.5,
			want:      0,
		},
		{
			name:      "2.x clamp high to 2.0",
			modelID:   "gemini-2.5-flash",
			requested: 3.0,
			want:      2.0,
		},
		{
			name:      "unknown model passes through",
			modelID:   "unknown-model",
			requested: 0.42,
			want:      0.42,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EnforceTemperature(tt.modelID, tt.requested)
			if got != tt.want {
				t.Errorf("EnforceTemperature(%q, %v) = %v, want %v", tt.modelID, tt.requested, got, tt.want)
			}
		})
	}
}

func TestApplyTemperature(t *testing.T) {
	req := &GenerateContentRequest{
		Contents: []Message{{Role: "user", Parts: []Part{{Text: "test"}}}},
	}

	ApplyTemperature(req, "gemini-3.1-pro-preview", 0.5)

	if req.GenerationConfig == nil {
		t.Fatal("GenerationConfig should be set")
	}
	if req.GenerationConfig.Temperature == nil {
		t.Fatal("Temperature should be set")
	}
	if *req.GenerationConfig.Temperature != 1.0 {
		t.Errorf("temperature = %v, want 1.0 (enforced for 3.x)", *req.GenerationConfig.Temperature)
	}
}

func TestApplyTemperature_ExistingConfig(t *testing.T) {
	maxTokens := 1000
	req := &GenerateContentRequest{
		GenerationConfig: &GenerationConfig{MaxOutputTokens: &maxTokens},
	}

	ApplyTemperature(req, "gemini-2.5-flash", 0.7)

	if *req.GenerationConfig.Temperature != 0.7 {
		t.Errorf("temperature = %v, want 0.7", *req.GenerationConfig.Temperature)
	}
	if *req.GenerationConfig.MaxOutputTokens != 1000 {
		t.Error("existing config field should be preserved")
	}
}

func TestValidateSystemPrompt(t *testing.T) {
	tests := []struct {
		name        string
		prompt      string
		modelID     string
		wantWarning bool
		wantContain string
	}{
		{
			name:        "clean prompt no warnings",
			prompt:      "You are a helpful assistant.",
			modelID:     "gemini-2.5-flash",
			wantWarning: false,
		},
		{
			name: "excessive negatives",
			prompt: "Never do X. Do not do Y. Don't do Z. Must not do A. " +
				"Never guess. Do not fabricate.",
			modelID:     "gemini-2.5-flash",
			wantWarning: true,
			wantContain: "negative constraints",
		},
		{
			name:        "very long prompt",
			prompt:      strings.Repeat("instruction ", 2000),
			modelID:     "",
			wantWarning: true,
			wantContain: "10,000 characters",
		},
		{
			name:        "temperature in text",
			prompt:      "Set temperature to 0.5 for creative tasks.",
			modelID:     "",
			wantWarning: true,
			wantContain: "GenerationConfig",
		},
		{
			name:        "JSON output without structured mode",
			prompt:      "Respond only in JSON format.",
			modelID:     "",
			wantWarning: true,
			wantContain: "EnableStructuredOutput",
		},
		{
			name:        "chain of thought on thinking model",
			prompt:      "Think step by step before answering.",
			modelID:     "gemini-3.1-pro-preview",
			wantWarning: true,
			wantContain: "ThinkingConfig",
		},
		{
			name:        "chain of thought on non-thinking model is fine",
			prompt:      "Think step by step before answering.",
			modelID:     "gemini-3.1-flash-image-preview",
			wantWarning: false, // Image model does not support thinking
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			warnings := ValidateSystemPrompt(tt.prompt, tt.modelID)
			if tt.wantWarning && len(warnings) == 0 {
				t.Error("expected warnings, got none")
			}
			if !tt.wantWarning && len(warnings) > 0 {
				t.Errorf("expected no warnings, got: %v", warnings)
			}
			if tt.wantContain != "" {
				found := false
				for _, w := range warnings {
					if strings.Contains(w, tt.wantContain) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected warning containing %q, got: %v", tt.wantContain, warnings)
				}
			}
		})
	}
}
