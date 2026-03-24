package gemini

import "fmt"

// SyntheticThoughtSignature is the constant used to bypass thought validation
// in cases where thoughts need to be injected synthetically.
const SyntheticThoughtSignature = "skip_thought_signature_validator"

// IsThinkingModel is defined in models.go.

// ExtractThoughtParts extracts thought parts from a Gemini response.
// Thought parts have Thought=true and non-empty Text.
func ExtractThoughtParts(resp *GenerateContentResponse) []ThoughtPart {
	if resp == nil || len(resp.Candidates) == 0 {
		return nil
	}

	content := resp.Candidates[0].Content
	if content == nil {
		return nil
	}

	var thoughts []ThoughtPart
	for _, part := range content.Parts {
		if part.Thought && part.Text != "" {
			thoughts = append(thoughts, ThoughtPart{
				Text:    part.Text,
				Thought: true,
			})
		}
	}
	return thoughts
}

// ExtractTextParts extracts non-thought text parts from a Gemini response.
func ExtractTextParts(resp *GenerateContentResponse) []string {
	if resp == nil || len(resp.Candidates) == 0 {
		return nil
	}

	content := resp.Candidates[0].Content
	if content == nil {
		return nil
	}

	var texts []string
	for _, part := range content.Parts {
		if !part.Thought && part.Text != "" {
			texts = append(texts, part.Text)
		}
	}
	return texts
}

// ValidateThoughtPart ensures a thought part meets Gemini 3.x requirements:
// - Text is non-empty
// - Thought flag is true
// - No function call/response data present
func ValidateThoughtPart(part Part) error {
	if !part.Thought && part.ThoughtSignature == "" {
		return fmt.Errorf("part must have thought=true or a thoughtSignature")
	}
	// Empty-text parts with a signature are valid (signature-only chunks).
	if part.Text == "" && part.ThoughtSignature == "" {
		return fmt.Errorf("thought part must have text or a signature")
	}
	if part.FunctionCall != nil {
		return fmt.Errorf("thought part must not contain function call")
	}
	if part.FunctionResponse != nil {
		return fmt.Errorf("thought part must not contain function response")
	}
	return nil
}

// CirculateThoughts preserves the model's response parts (including thought
// signatures) for the next turn. Gemini 3.x requires that model response parts
// are returned EXACTLY as received -- never concatenate, merge, or reconstruct.
//
// The correct pattern is to include the model's full response (with all its
// original parts including thoughtSignature fields) as a "model" message in
// the conversation history. This function takes the original model parts
// and ensures they are preserved in the message history.
func CirculateThoughts(messages []Message, originalModelParts []Part) []Message {
	if len(originalModelParts) == 0 {
		return messages
	}

	result := append([]Message(nil), messages...)

	// Find the last model message and replace its parts with the originals.
	for i := len(result) - 1; i >= 0; i-- {
		if result[i].Role == "model" {
			result[i].Parts = originalModelParts
			return result
		}
	}

	// No model message found -- append one with the original parts.
	result = append(result, Message{
		Role:  "model",
		Parts: originalModelParts,
	})
	return result
}

// NewThinkingConfig creates a ThinkingConfig for controlling reasoning.
// Level: ThinkingLevelLow, ThinkingLevelMedium, ThinkingLevelHigh, ThinkingLevelMinimal
// Budget: maximum thinking tokens (0 = model default, -1 = dynamic)
// includeThoughts must be true to receive thought text in responses.
func NewThinkingConfig(level string, budget int, includeThoughts bool) *ThinkingConfig {
	return &ThinkingConfig{
		IncludeThoughts: includeThoughts,
		ThinkingLevel:   level,
		ThinkingBudget:  budget,
	}
}

// ApplyThinkingConfig sets thinking parameters on a generation request.
// Only applies if the target model supports thinking.
func ApplyThinkingConfig(request *GenerateContentRequest, model string, config *ThinkingConfig) {
	if config == nil || !IsThinkingModel(model) {
		return
	}
	if request.GenerationConfig == nil {
		request.GenerationConfig = &GenerationConfig{}
	}
	request.GenerationConfig.ThinkingConfig = config
}
