package gemini

// Safety harm categories as defined by the Gemini API.
const (
	HarmCategoryHarassment        = "HARM_CATEGORY_HARASSMENT"
	HarmCategoryHateSpeech        = "HARM_CATEGORY_HATE_SPEECH"
	HarmCategoryDangerousContent  = "HARM_CATEGORY_DANGEROUS_CONTENT"
	HarmCategorySexuallyExplicit  = "HARM_CATEGORY_SEXUALLY_EXPLICIT"
	HarmCategoryCivicIntegrity    = "HARM_CATEGORY_CIVIC_INTEGRITY"
)

// Safety threshold levels.
const (
	BlockNone           = "BLOCK_NONE"
	BlockOnlyHigh       = "BLOCK_ONLY_HIGH"
	BlockMediumAndAbove = "BLOCK_MEDIUM_AND_ABOVE"
	BlockLowAndAbove    = "BLOCK_LOW_AND_ABOVE"
)

// AllHarmCategories returns all supported harm categories.
func AllHarmCategories() []string {
	return []string{
		HarmCategoryHarassment,
		HarmCategoryHateSpeech,
		HarmCategoryDangerousContent,
		HarmCategorySexuallyExplicit,
		HarmCategoryCivicIntegrity,
	}
}

// DefaultSafetySettings returns safety settings optimised for technical/coding
// content. All categories set to BlockOnlyHigh to avoid false positives when
// processing technical documentation, code snippets, or research content.
func DefaultSafetySettings() []SafetySetting {
	return buildSettings(BlockOnlyHigh)
}

// PermissiveSafetySettings returns BlockNone for all categories.
// Use with caution -- only appropriate for controlled/internal environments.
func PermissiveSafetySettings() []SafetySetting {
	return buildSettings(BlockNone)
}

// StrictSafetySettings returns BlockLowAndAbove for all categories.
// Most restrictive configuration.
func StrictSafetySettings() []SafetySetting {
	return buildSettings(BlockLowAndAbove)
}

// CustomSafety creates safety settings with per-category overrides.
// Categories not in the overrides map use the fallback threshold.
func CustomSafety(overrides map[string]string, fallback string) []SafetySetting {
	var settings []SafetySetting
	for _, cat := range AllHarmCategories() {
		threshold := fallback
		if t, ok := overrides[cat]; ok {
			threshold = t
		}
		settings = append(settings, SafetySetting{
			Category:  cat,
			Threshold: threshold,
		})
	}
	return settings
}

// ApplySafetySettings sets safety settings on a request.
func ApplySafetySettings(request *GenerateContentRequest, settings []SafetySetting) {
	request.SafetySettings = settings
}

// ValidateResponse checks if a Gemini response was blocked by safety filters.
// Returns true if blocked, along with the blocking reasons.
func ValidateResponse(resp *GenerateContentResponse) (blocked bool, reasons []string) {
	if resp == nil {
		return false, nil
	}

	// Check prompt-level blocking.
	if resp.PromptFeedback != nil && resp.PromptFeedback.BlockReason != "" {
		reasons = append(reasons, "prompt:"+resp.PromptFeedback.BlockReason)
		return true, reasons
	}

	// Check candidate-level blocking.
	if len(resp.Candidates) > 0 {
		c := resp.Candidates[0]
		if c.FinishReason == "SAFETY" {
			for _, r := range c.SafetyRatings {
				if r.Blocked {
					reasons = append(reasons, r.Category)
				}
			}
			return true, reasons
		}
	}

	return false, nil
}

// buildSettings creates SafetySettings with the same threshold for all categories.
func buildSettings(threshold string) []SafetySetting {
	var settings []SafetySetting
	for _, cat := range AllHarmCategories() {
		settings = append(settings, SafetySetting{
			Category:  cat,
			Threshold: threshold,
		})
	}
	return settings
}
