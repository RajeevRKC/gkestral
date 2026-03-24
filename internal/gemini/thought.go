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
	if !part.Thought {
		return fmt.Errorf("thought flag must be true")
	}
	if part.Text == "" {
		return fmt.Errorf("thought text must be non-empty")
	}
	if part.FunctionCall != nil {
		return fmt.Errorf("thought part must not contain function call")
	}
	if part.FunctionResponse != nil {
		return fmt.Errorf("thought part must not contain function response")
	}
	if part.InlineData != nil {
		return fmt.Errorf("thought part must not contain inline data")
	}
	return nil
}

// CirculateThoughts injects thought parts back into the conversation history
// for subsequent turns. Gemini 3.x requires thoughts from the model's previous
// response to be included in the next request for coherent reasoning.
func CirculateThoughts(messages []Message, thoughts []ThoughtPart) []Message {
	if len(thoughts) == 0 {
		return messages
	}

	// Build model message with thought parts first, then any existing parts.
	var thoughtParts []Part
	for _, t := range thoughts {
		thoughtParts = append(thoughtParts, Part{
			Text:    t.Text,
			Thought: true,
		})
	}

	// Find the last model message and prepend thoughts.
	result := make([]Message, len(messages))
	copy(result, messages)

	for i := len(result) - 1; i >= 0; i-- {
		if result[i].Role == "model" {
			result[i].Parts = append(thoughtParts, result[i].Parts...)
			return result
		}
	}

	// No model message found -- append a new one with just thoughts.
	result = append(result, Message{
		Role:  "model",
		Parts: thoughtParts,
	})
	return result
}

// NewThinkingConfig creates a ThinkingConfig for controlling reasoning.
// Level: "off", "low", "medium", "high"
// Budget: maximum thinking tokens (0 = model default)
func NewThinkingConfig(level string, budget int) *ThinkingConfig {
	return &ThinkingConfig{
		ThinkingLevel:  level,
		ThinkingBudget: budget,
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
