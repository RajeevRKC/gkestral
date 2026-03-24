package gemini

import (
	"fmt"
	"sort"
)

// ModelFamily classifies models into capability groups.
type ModelFamily int

const (
	// FamilyPro31 is the Gemini 3.1 Pro family (deep reasoning, architecture).
	FamilyPro31 ModelFamily = iota
	// FamilyFlash31 is the Gemini 3.1 Flash family (fast, high-volume).
	FamilyFlash31
	// FamilyFlash31Image is the Gemini 3.1 Flash image generation variant.
	FamilyFlash31Image
	// FamilyPro25 is the Gemini 2.5 Pro family (production stable).
	FamilyPro25
	// FamilyFlash25 is the Gemini 2.5 Flash family (production stable, cheap).
	FamilyFlash25
)

// ModelConfig holds the full specification for a Gemini model.
type ModelConfig struct {
	ID               string      `json:"id"`
	DisplayName      string      `json:"displayName"`
	Family           ModelFamily `json:"family"`
	InputPricePerM   float64     `json:"inputPricePerMillion"`   // USD per 1M input tokens
	OutputPricePerM  float64     `json:"outputPricePerMillion"`  // USD per 1M output tokens
	CachingDiscount  float64     `json:"cachingDiscount"`        // Fraction (0.0-1.0), e.g., 0.75 = 75% discount
	ContextWindow    int         `json:"contextWindow"`          // Max input tokens
	MaxOutput        int         `json:"maxOutput"`              // Max output tokens
	SupportsThinking bool        `json:"supportsThinking"`
	SupportsGrounding bool       `json:"supportsGrounding"`
	SupportsCaching  bool        `json:"supportsCaching"`
	MinCacheTokens   int         `json:"minCacheTokens"`         // Minimum tokens to create a cache
	ThinkingDefault  string      `json:"thinkingDefault"`        // Default thinking level
}

// modelRegistry contains configurations for all target models.
// Pricing reflects Gemini API as of March 2026.
var modelRegistry = map[string]ModelConfig{
	"gemini-3.1-pro-preview": {
		ID:               "gemini-3.1-pro-preview",
		DisplayName:      "Gemini 3.1 Pro",
		Family:           FamilyPro31,
		InputPricePerM:   1.25,
		OutputPricePerM:  10.00,
		CachingDiscount:  0.75,
		ContextWindow:    1_048_576,
		MaxOutput:        65_536,
		SupportsThinking: true,
		SupportsGrounding: true,
		SupportsCaching:  true,
		MinCacheTokens:   4096,
		ThinkingDefault:  "medium",
	},
	"gemini-3.1-flash": {
		ID:               "gemini-3.1-flash",
		DisplayName:      "Gemini 3.1 Flash",
		Family:           FamilyFlash31,
		InputPricePerM:   0.15,
		OutputPricePerM:  0.60,
		CachingDiscount:  0.75,
		ContextWindow:    1_048_576,
		MaxOutput:        8_192,
		SupportsThinking: true,
		SupportsGrounding: true,
		SupportsCaching:  true,
		MinCacheTokens:   1024,
		ThinkingDefault:  "low",
	},
	"gemini-3.1-flash-image-preview": {
		ID:               "gemini-3.1-flash-image-preview",
		DisplayName:      "Gemini 3.1 Flash (Image/NanoBanana Pro)",
		Family:           FamilyFlash31Image,
		InputPricePerM:   0.15,
		OutputPricePerM:  0.60,
		CachingDiscount:  0.0, // No caching for image gen
		ContextWindow:    1_048_576,
		MaxOutput:        8_192,
		SupportsThinking: false,
		SupportsGrounding: false,
		SupportsCaching:  false,
		MinCacheTokens:   0,
		ThinkingDefault:  "off",
	},
	"gemini-2.5-pro": {
		ID:               "gemini-2.5-pro",
		DisplayName:      "Gemini 2.5 Pro",
		Family:           FamilyPro25,
		InputPricePerM:   1.25,
		OutputPricePerM:  10.00,
		CachingDiscount:  0.75,
		ContextWindow:    1_048_576,
		MaxOutput:        65_536,
		SupportsThinking: true,
		SupportsGrounding: true,
		SupportsCaching:  true,
		MinCacheTokens:   4096,
		ThinkingDefault:  "off",
	},
	"gemini-2.5-flash": {
		ID:               "gemini-2.5-flash",
		DisplayName:      "Gemini 2.5 Flash",
		Family:           FamilyFlash25,
		InputPricePerM:   0.15,
		OutputPricePerM:  0.60,
		CachingDiscount:  0.75,
		ContextWindow:    1_048_576,
		MaxOutput:        8_192,
		SupportsThinking: true,
		SupportsGrounding: true,
		SupportsCaching:  true,
		MinCacheTokens:   1024,
		ThinkingDefault:  "off",
	},
}

// GetModel returns the configuration for the specified model ID.
// Returns an error if the model is not in the registry.
func GetModel(id string) (ModelConfig, error) {
	model, ok := modelRegistry[id]
	if !ok {
		return ModelConfig{}, fmt.Errorf("unknown model: %q", id)
	}
	return model, nil
}

// ListModels returns all registered model configurations, sorted by ID.
func ListModels() []ModelConfig {
	models := make([]ModelConfig, 0, len(modelRegistry))
	for _, m := range modelRegistry {
		models = append(models, m)
	}
	sort.Slice(models, func(i, j int) bool {
		return models[i].ID < models[j].ID
	})
	return models
}

// TokenEconomics calculates the cost for a Gemini API call with and without caching.
func TokenEconomics(modelID string, inputTokens, outputTokens, cachedTokens int) (CostEstimate, error) {
	model, err := GetModel(modelID)
	if err != nil {
		return CostEstimate{}, err
	}

	// Standard costs (per million tokens)
	uncachedInput := float64(inputTokens) / 1_000_000.0 * model.InputPricePerM
	outputCost := float64(outputTokens) / 1_000_000.0 * model.OutputPricePerM

	// Cached costs: cached tokens get the discount, uncached tokens pay full price
	nonCachedInput := inputTokens - cachedTokens
	if nonCachedInput < 0 {
		nonCachedInput = 0
	}

	cachedInputCost := float64(nonCachedInput) / 1_000_000.0 * model.InputPricePerM
	if cachedTokens > 0 {
		// Cached tokens get the discount rate. If no discount (CachingDiscount == 0),
		// they are billed at full input price -- never free.
		discountedRate := model.InputPricePerM * (1.0 - model.CachingDiscount)
		cachedInputCost += float64(cachedTokens) / 1_000_000.0 * discountedRate
	}

	totalWithoutCache := uncachedInput + outputCost
	totalWithCache := cachedInputCost + outputCost

	return CostEstimate{
		InputCost:       uncachedInput,
		OutputCost:      outputCost,
		CachedInputCost: cachedInputCost,
		TotalCost:       totalWithCache,
		Savings:         totalWithoutCache - totalWithCache,
	}, nil
}

// IsThinkingModel returns true if the model supports the thinking/reasoning mode.
func IsThinkingModel(modelID string) bool {
	model, err := GetModel(modelID)
	if err != nil {
		return false
	}
	return model.SupportsThinking
}
