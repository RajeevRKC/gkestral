package gemini

import (
	"math"
	"testing"
)

func TestGetModel_KnownModels(t *testing.T) {
	tests := []struct {
		name    string
		modelID string
		wantErr bool
	}{
		{name: "3.1 pro", modelID: "gemini-3.1-pro-preview", wantErr: false},
		{name: "3.1 flash", modelID: "gemini-3.1-flash", wantErr: false},
		{name: "3.1 flash image", modelID: "gemini-3.1-flash-image-preview", wantErr: false},
		{name: "2.5 pro", modelID: "gemini-2.5-pro", wantErr: false},
		{name: "2.5 flash", modelID: "gemini-2.5-flash", wantErr: false},
		{name: "unknown", modelID: "gemini-99.9-ultra", wantErr: true},
		{name: "empty", modelID: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model, err := GetModel(tt.modelID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetModel(%q) error = %v, wantErr %v", tt.modelID, err, tt.wantErr)
				return
			}
			if !tt.wantErr && model.ID != tt.modelID {
				t.Errorf("GetModel(%q).ID = %q, want %q", tt.modelID, model.ID, tt.modelID)
			}
		})
	}
}

func TestGetModel_Capabilities(t *testing.T) {
	tests := []struct {
		modelID           string
		wantThinking      bool
		wantGrounding     bool
		wantCaching       bool
		wantMinCache      int
		wantContextWindow int
	}{
		{
			modelID:           "gemini-3.1-pro-preview",
			wantThinking:      true,
			wantGrounding:     true,
			wantCaching:       true,
			wantMinCache:      4096,
			wantContextWindow: 1_000_000,
		},
		{
			modelID:           "gemini-3.1-flash",
			wantThinking:      true,
			wantGrounding:     true,
			wantCaching:       true,
			wantMinCache:      1024,
			wantContextWindow: 1_000_000,
		},
		{
			modelID:           "gemini-3.1-flash-image-preview",
			wantThinking:      false,
			wantGrounding:     false,
			wantCaching:       false,
			wantMinCache:      0,
			wantContextWindow: 1_000_000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.modelID, func(t *testing.T) {
			m, err := GetModel(tt.modelID)
			if err != nil {
				t.Fatalf("GetModel(%q) unexpected error: %v", tt.modelID, err)
			}
			if m.SupportsThinking != tt.wantThinking {
				t.Errorf("SupportsThinking = %v, want %v", m.SupportsThinking, tt.wantThinking)
			}
			if m.SupportsGrounding != tt.wantGrounding {
				t.Errorf("SupportsGrounding = %v, want %v", m.SupportsGrounding, tt.wantGrounding)
			}
			if m.SupportsCaching != tt.wantCaching {
				t.Errorf("SupportsCaching = %v, want %v", m.SupportsCaching, tt.wantCaching)
			}
			if m.MinCacheTokens != tt.wantMinCache {
				t.Errorf("MinCacheTokens = %d, want %d", m.MinCacheTokens, tt.wantMinCache)
			}
			if m.ContextWindow != tt.wantContextWindow {
				t.Errorf("ContextWindow = %d, want %d", m.ContextWindow, tt.wantContextWindow)
			}
		})
	}
}

func TestListModels(t *testing.T) {
	models := ListModels()
	if len(models) != 5 {
		t.Errorf("ListModels() returned %d models, want 5", len(models))
	}

	// Verify sorted by ID
	for i := 1; i < len(models); i++ {
		if models[i].ID < models[i-1].ID {
			t.Errorf("ListModels() not sorted: %q < %q at index %d", models[i].ID, models[i-1].ID, i)
		}
	}
}

func TestTokenEconomics_BasicCost(t *testing.T) {
	// 1M input tokens + 1M output tokens on 2.5-flash
	est, err := TokenEconomics("gemini-2.5-flash", 1_000_000, 1_000_000, 0)
	if err != nil {
		t.Fatalf("TokenEconomics unexpected error: %v", err)
	}

	// Input: 1M * $0.15/M = $0.15
	if !approxEqual(est.InputCost, 0.15, 0.001) {
		t.Errorf("InputCost = %f, want ~0.15", est.InputCost)
	}
	// Output: 1M * $0.60/M = $0.60
	if !approxEqual(est.OutputCost, 0.60, 0.001) {
		t.Errorf("OutputCost = %f, want ~0.60", est.OutputCost)
	}
	// No caching, savings should be 0
	if !approxEqual(est.Savings, 0.0, 0.001) {
		t.Errorf("Savings = %f, want ~0.0", est.Savings)
	}
}

func TestTokenEconomics_WithCaching(t *testing.T) {
	// 1M input (800K cached, 200K fresh) + 100K output on 2.5-flash
	est, err := TokenEconomics("gemini-2.5-flash", 1_000_000, 100_000, 800_000)
	if err != nil {
		t.Fatalf("TokenEconomics unexpected error: %v", err)
	}

	// Without cache: 1M * $0.15/M = $0.15
	// With cache: 200K * $0.15/M + 800K * ($0.15 * 0.25)/M
	//           = 200K * 0.15/1M + 800K * 0.0375/1M
	//           = 0.03 + 0.03 = 0.06
	// Savings = 0.15 - 0.06 = 0.09
	if est.Savings <= 0 {
		t.Errorf("Expected positive savings with caching, got %f", est.Savings)
	}
	if !approxEqual(est.Savings, 0.09, 0.01) {
		t.Errorf("Savings = %f, want ~0.09", est.Savings)
	}
}

func TestTokenEconomics_UnknownModel(t *testing.T) {
	_, err := TokenEconomics("gemini-99.9-ultra", 1000, 1000, 0)
	if err == nil {
		t.Error("Expected error for unknown model, got nil")
	}
}

func TestIsThinkingModel(t *testing.T) {
	tests := []struct {
		modelID string
		want    bool
	}{
		{"gemini-3.1-pro-preview", true},
		{"gemini-3.1-flash", true},
		{"gemini-3.1-flash-image-preview", false},
		{"gemini-2.5-pro", true},
		{"gemini-2.5-flash", true},
		{"unknown-model", false},
	}

	for _, tt := range tests {
		t.Run(tt.modelID, func(t *testing.T) {
			got := IsThinkingModel(tt.modelID)
			if got != tt.want {
				t.Errorf("IsThinkingModel(%q) = %v, want %v", tt.modelID, got, tt.want)
			}
		})
	}
}

func approxEqual(a, b, epsilon float64) bool {
	return math.Abs(a-b) < epsilon
}
