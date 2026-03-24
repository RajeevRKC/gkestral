package gemini

import "testing"

// TestIsThinkingModel is in models_test.go.

func TestExtractThoughtParts(t *testing.T) {
	resp := &GenerateContentResponse{
		Candidates: []Candidate{{
			Content: &CandidateContent{
				Role: "model",
				Parts: []Part{
					{Text: "I need to think about this...", Thought: true},
					{Text: "The answer is 42"},
					{Text: "Also considering...", Thought: true},
				},
			},
		}},
	}

	thoughts := ExtractThoughtParts(resp)
	if len(thoughts) != 2 {
		t.Fatalf("want 2 thoughts, got %d", len(thoughts))
	}
	if thoughts[0].Text != "I need to think about this..." {
		t.Errorf("thought 0: got %q", thoughts[0].Text)
	}
	if thoughts[1].Text != "Also considering..." {
		t.Errorf("thought 1: got %q", thoughts[1].Text)
	}
}

func TestExtractThoughtParts_NilResponse(t *testing.T) {
	if tp := ExtractThoughtParts(nil); tp != nil {
		t.Errorf("expected nil for nil response, got %v", tp)
	}
	if tp := ExtractThoughtParts(&GenerateContentResponse{}); tp != nil {
		t.Errorf("expected nil for empty response, got %v", tp)
	}
}

func TestExtractTextParts(t *testing.T) {
	resp := &GenerateContentResponse{
		Candidates: []Candidate{{
			Content: &CandidateContent{
				Role: "model",
				Parts: []Part{
					{Text: "thinking...", Thought: true},
					{Text: "Hello"},
					{Text: "World"},
				},
			},
		}},
	}

	texts := ExtractTextParts(resp)
	if len(texts) != 2 {
		t.Fatalf("want 2 texts, got %d", len(texts))
	}
	if texts[0] != "Hello" || texts[1] != "World" {
		t.Errorf("texts: got %v", texts)
	}
}

func TestValidateThoughtPart(t *testing.T) {
	tests := []struct {
		name    string
		part    Part
		wantErr bool
	}{
		{"valid_thought", Part{Text: "thinking...", Thought: true}, false},
		{"valid_signature_only", Part{ThoughtSignature: "abc123"}, false},
		{"valid_thought_with_sig", Part{Text: "t", Thought: true, ThoughtSignature: "sig"}, false},
		{"no_thought_no_sig", Part{Text: "text"}, true},
		{"empty_text_no_sig", Part{Thought: true}, true},
		{"has_function_call", Part{Text: "t", Thought: true, FunctionCall: &FunctionCall{Name: "fn"}}, true},
		{"has_function_response", Part{Text: "t", Thought: true, FunctionResponse: &FunctionResponse{Name: "fn"}}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateThoughtPart(tt.part)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateThoughtPart() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCirculateThoughts_PreservesOriginalParts(t *testing.T) {
	messages := []Message{
		{Role: "user", Parts: []Part{{Text: "What is 6*7?"}}},
		{Role: "model", Parts: []Part{{Text: "old response"}}},
		{Role: "user", Parts: []Part{{Text: "Explain"}}},
	}

	// Original parts from the model response -- must be preserved exactly.
	originalParts := []Part{
		{Text: "thinking...", Thought: true, ThoughtSignature: "abc123sig"},
		{Text: "42"},
	}

	result := CirculateThoughts(messages, originalParts)
	if len(result) != 3 {
		t.Fatalf("want 3 messages, got %d", len(result))
	}

	modelMsg := result[1]
	if len(modelMsg.Parts) != 2 {
		t.Fatalf("model message: want 2 parts, got %d", len(modelMsg.Parts))
	}
	if modelMsg.Parts[0].ThoughtSignature != "abc123sig" {
		t.Error("thought signature must be preserved exactly")
	}
	if modelMsg.Parts[1].Text != "42" {
		t.Errorf("second part should be '42', got %q", modelMsg.Parts[1].Text)
	}
}

func TestCirculateThoughts_Empty(t *testing.T) {
	messages := []Message{{Role: "user", Parts: []Part{{Text: "Hi"}}}}
	result := CirculateThoughts(messages, nil)
	if len(result) != 1 {
		t.Errorf("nil parts should return original messages, got %d", len(result))
	}
}

func TestCirculateThoughts_NoModelMessage(t *testing.T) {
	messages := []Message{{Role: "user", Parts: []Part{{Text: "Hi"}}}}
	originalParts := []Part{{Text: "thinking", Thought: true, ThoughtSignature: "sig"}}

	result := CirculateThoughts(messages, originalParts)
	if len(result) != 2 {
		t.Fatalf("want 2 messages, got %d", len(result))
	}
	if result[1].Role != "model" {
		t.Errorf("new message role: want 'model', got %q", result[1].Role)
	}
	if result[1].Parts[0].ThoughtSignature != "sig" {
		t.Error("signature must be preserved on appended model message")
	}
}

func TestNewThinkingConfig(t *testing.T) {
	cfg := NewThinkingConfig(ThinkingLevelHigh, 8192, true)
	if cfg.ThinkingLevel != ThinkingLevelHigh {
		t.Errorf("level: want %q, got %q", ThinkingLevelHigh, cfg.ThinkingLevel)
	}
	if cfg.ThinkingBudget != 8192 {
		t.Errorf("budget: want 8192, got %d", cfg.ThinkingBudget)
	}
	if !cfg.IncludeThoughts {
		t.Error("includeThoughts should be true")
	}
}

func TestApplyThinkingConfig(t *testing.T) {
	req := &GenerateContentRequest{
		Contents: []Message{{Role: "user", Parts: []Part{{Text: "Hi"}}}},
	}

	// Should apply for thinking model.
	cfg := NewThinkingConfig(ThinkingLevelMedium, 4096, true)
	ApplyThinkingConfig(req, "gemini-3.1-pro-preview", cfg)
	if req.GenerationConfig == nil || req.GenerationConfig.ThinkingConfig == nil {
		t.Fatal("thinking config should be applied for thinking model")
	}
	if req.GenerationConfig.ThinkingConfig.ThinkingLevel != ThinkingLevelMedium {
		t.Errorf("level: want %q, got %q", ThinkingLevelMedium, req.GenerationConfig.ThinkingConfig.ThinkingLevel)
	}
	if !req.GenerationConfig.ThinkingConfig.IncludeThoughts {
		t.Error("includeThoughts should be true")
	}

	// Should not apply for non-thinking model.
	req2 := &GenerateContentRequest{}
	ApplyThinkingConfig(req2, "gemini-2.5-flash-lite", cfg)
	if req2.GenerationConfig != nil {
		t.Error("thinking config should NOT be applied for non-thinking model")
	}

	// Should handle nil config.
	req3 := &GenerateContentRequest{}
	ApplyThinkingConfig(req3, "gemini-3.1-pro-preview", nil)
	if req3.GenerationConfig != nil {
		t.Error("nil config should not create GenerationConfig")
	}
}
