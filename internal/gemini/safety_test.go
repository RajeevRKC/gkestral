package gemini

import (
	"encoding/json"
	"testing"
)

func TestDefaultSafetySettings(t *testing.T) {
	settings := DefaultSafetySettings()
	if len(settings) != 5 {
		t.Fatalf("want 5 settings, got %d", len(settings))
	}
	for _, s := range settings {
		if s.Threshold != BlockOnlyHigh {
			t.Errorf("category %s: want %s, got %s", s.Category, BlockOnlyHigh, s.Threshold)
		}
	}
}

func TestPermissiveSafetySettings(t *testing.T) {
	settings := PermissiveSafetySettings()
	for _, s := range settings {
		if s.Threshold != BlockNone {
			t.Errorf("category %s: want %s, got %s", s.Category, BlockNone, s.Threshold)
		}
	}
}

func TestStrictSafetySettings(t *testing.T) {
	settings := StrictSafetySettings()
	for _, s := range settings {
		if s.Threshold != BlockLowAndAbove {
			t.Errorf("category %s: want %s, got %s", s.Category, BlockLowAndAbove, s.Threshold)
		}
	}
}

func TestCustomSafety(t *testing.T) {
	overrides := map[string]string{
		HarmCategoryHarassment: BlockNone,
		HarmCategoryHateSpeech: BlockMediumAndAbove,
	}
	settings := CustomSafety(overrides, BlockOnlyHigh)

	if len(settings) != 5 {
		t.Fatalf("want 5 settings, got %d", len(settings))
	}

	for _, s := range settings {
		switch s.Category {
		case HarmCategoryHarassment:
			if s.Threshold != BlockNone {
				t.Errorf("harassment: want %s, got %s", BlockNone, s.Threshold)
			}
		case HarmCategoryHateSpeech:
			if s.Threshold != BlockMediumAndAbove {
				t.Errorf("hate: want %s, got %s", BlockMediumAndAbove, s.Threshold)
			}
		default:
			if s.Threshold != BlockOnlyHigh {
				t.Errorf("%s: want fallback %s, got %s", s.Category, BlockOnlyHigh, s.Threshold)
			}
		}
	}
}

func TestSafetySettingsJSON(t *testing.T) {
	settings := DefaultSafetySettings()
	data, err := json.Marshal(settings)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var parsed []SafetySetting
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if len(parsed) != 5 {
		t.Errorf("round-trip: want 5, got %d", len(parsed))
	}
}

func TestValidateResponse_NotBlocked(t *testing.T) {
	resp := &GenerateContentResponse{
		Candidates: []Candidate{{
			Content:      &CandidateContent{Role: "model", Parts: []Part{{Text: "Hello"}}},
			FinishReason: "STOP",
		}},
	}
	blocked, reasons := ValidateResponse(resp)
	if blocked {
		t.Errorf("expected not blocked, got reasons: %v", reasons)
	}
}

func TestValidateResponse_CandidateBlocked(t *testing.T) {
	resp := &GenerateContentResponse{
		Candidates: []Candidate{{
			FinishReason: "SAFETY",
			SafetyRatings: []SafetyRating{
				{Category: HarmCategoryDangerousContent, Probability: "HIGH", Blocked: true},
				{Category: HarmCategoryHarassment, Probability: "LOW", Blocked: false},
			},
		}},
	}
	blocked, reasons := ValidateResponse(resp)
	if !blocked {
		t.Fatal("expected blocked")
	}
	if len(reasons) != 1 || reasons[0] != HarmCategoryDangerousContent {
		t.Errorf("reasons: want [%s], got %v", HarmCategoryDangerousContent, reasons)
	}
}

func TestValidateResponse_PromptBlocked(t *testing.T) {
	resp := &GenerateContentResponse{
		PromptFeedback: &PromptFeedback{
			BlockReason: "SAFETY",
		},
	}
	blocked, reasons := ValidateResponse(resp)
	if !blocked {
		t.Fatal("expected blocked")
	}
	if len(reasons) != 1 || reasons[0] != "prompt:SAFETY" {
		t.Errorf("reasons: want [prompt:SAFETY], got %v", reasons)
	}
}

func TestValidateResponse_Nil(t *testing.T) {
	blocked, _ := ValidateResponse(nil)
	if blocked {
		t.Error("nil response should not be blocked")
	}
}

func TestApplySafetySettings(t *testing.T) {
	req := &GenerateContentRequest{}
	settings := DefaultSafetySettings()
	ApplySafetySettings(req, settings)
	if len(req.SafetySettings) != 5 {
		t.Errorf("want 5 safety settings applied, got %d", len(req.SafetySettings))
	}
}
