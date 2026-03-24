package gemini

import (
	"fmt"
	"strings"
)

// SystemPrompt is a structured representation of a system instruction that
// can be compiled into a Gemini-compatible Message. Fields are concatenated
// in a well-tested order that works best with Gemini models.
type SystemPrompt struct {
	// Role defines who or what the assistant should be.
	Role string
	// Constraints are hard rules the model must follow.
	Constraints []string
	// OutputFormat describes how the model should format its responses.
	OutputFormat string
	// ToolGuidance explains when and how to use available tools.
	ToolGuidance string
	// CustomInstructions is free-form additional instructions.
	CustomInstructions string
}

// BuildSystemInstruction compiles a SystemPrompt into a *Message suitable
// for the systemInstruction field of a GenerateContentRequest.
//
// The Gemini API expects systemInstruction as a Content object (Message)
// with role "user" and a single text Part, NOT a bare Part or string.
func BuildSystemInstruction(prompt SystemPrompt) *Message {
	var sb strings.Builder

	if prompt.Role != "" {
		sb.WriteString(prompt.Role)
		sb.WriteString("\n\n")
	}

	if len(prompt.Constraints) > 0 {
		sb.WriteString("## Constraints\n")
		for _, c := range prompt.Constraints {
			sb.WriteString("- ")
			sb.WriteString(c)
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	if prompt.OutputFormat != "" {
		sb.WriteString("## Output Format\n")
		sb.WriteString(prompt.OutputFormat)
		sb.WriteString("\n\n")
	}

	if prompt.ToolGuidance != "" {
		sb.WriteString("## Tool Usage\n")
		sb.WriteString(prompt.ToolGuidance)
		sb.WriteString("\n\n")
	}

	if prompt.CustomInstructions != "" {
		sb.WriteString(prompt.CustomInstructions)
		sb.WriteString("\n")
	}

	text := strings.TrimSpace(sb.String())
	if text == "" {
		text = "You are a helpful assistant."
	}

	return &Message{
		Role: "user",
		Parts: []Part{
			{Text: text},
		},
	}
}

// BuildSystemInstructionFromText creates a *Message from a plain text string.
// Use when you have a pre-formatted system prompt.
func BuildSystemInstructionFromText(text string) *Message {
	if text == "" {
		text = "You are a helpful assistant."
	}
	return &Message{
		Role: "user",
		Parts: []Part{
			{Text: text},
		},
	}
}

// ApplySystemInstruction sets the systemInstruction field on a request.
func ApplySystemInstruction(request *GenerateContentRequest, instruction *Message) {
	request.SystemInstruction = instruction
}

// DefaultSystemPrompt returns a system prompt optimised for Gemini's behaviour
// with technical and productivity tasks.
func DefaultSystemPrompt() SystemPrompt {
	return SystemPrompt{
		Role: "You are a precise and helpful assistant. You provide clear, " +
			"well-structured responses. When working with code, you follow " +
			"language conventions and best practices.",
		Constraints: []string{
			"Be concise -- avoid unnecessary preamble or filler",
			"When uncertain, state your confidence level explicitly",
			"If a task requires tools, use them proactively rather than describing what you would do",
			"Preserve exact formatting of code, configuration, and data structures",
			"Never fabricate citations, URLs, or factual claims -- use search grounding when needed",
		},
		OutputFormat: "Use markdown for structured responses. Use code blocks with language " +
			"tags for code. Use bullet points for lists. Keep paragraphs short.",
		ToolGuidance: "Use tools when the task requires reading files, searching, or " +
			"performing actions. Prefer parallel tool calls when multiple " +
			"independent lookups are needed. Report tool errors clearly.",
	}
}

// EnforceTemperature returns the correct temperature value for a given model.
// Gemini 3.x models should always use 1.0 (lower values cause looping).
// Gemini 2.x models accept any value in [0.0, 2.0].
func EnforceTemperature(modelID string, requested float64) float64 {
	model, err := GetModel(modelID)
	if err != nil {
		return requested
	}

	// 3.x models: force 1.0 to prevent known looping issue.
	switch model.Family {
	case FamilyPro31, FamilyFlash31, FamilyFlash31Image:
		return 1.0
	}

	// 2.x models: clamp to valid range.
	if requested < 0 {
		return 0
	}
	if requested > 2.0 {
		return 2.0
	}
	return requested
}

// ApplyTemperature sets the temperature on a request, enforcing model-specific
// rules. Creates the GenerationConfig if nil.
func ApplyTemperature(request *GenerateContentRequest, modelID string, temperature float64) {
	enforced := EnforceTemperature(modelID, temperature)
	if request.GenerationConfig == nil {
		request.GenerationConfig = &GenerationConfig{}
	}
	request.GenerationConfig.Temperature = &enforced
}

// ValidateSystemPrompt checks a system prompt string for known patterns
// that cause issues with Gemini models. Returns warnings (not errors) --
// the prompt will still work but may produce suboptimal results.
func ValidateSystemPrompt(prompt string, modelID string) []string {
	var warnings []string
	lower := strings.ToLower(prompt)

	// Pattern: overly restrictive "never" / "do not" chains.
	negativeCount := strings.Count(lower, "never") +
		strings.Count(lower, "do not") +
		strings.Count(lower, "don't") +
		strings.Count(lower, "must not")
	if negativeCount > 5 {
		warnings = append(warnings,
			fmt.Sprintf("system prompt has %d negative constraints; excessive restrictions can cause Gemini to loop or refuse tasks -- consider rephrasing positively", negativeCount))
	}

	// Pattern: very long system prompts.
	if len(prompt) > 10000 {
		warnings = append(warnings,
			"system prompt exceeds 10,000 characters; consider moving reference material to context caching instead")
	}

	// Pattern: explicit temperature instructions in text (model will ignore).
	if strings.Contains(lower, "temperature") && strings.Contains(lower, "set") {
		warnings = append(warnings,
			"system prompt mentions temperature settings; temperature is controlled via GenerationConfig, not prompt text")
	}

	// Pattern: JSON-only instructions in the prompt text (should use structured output mode instead).
	if strings.Contains(lower, "respond only in json") || strings.Contains(lower, "output json only") ||
		strings.Contains(lower, "return json") || strings.Contains(lower, "reply in json") {
		warnings = append(warnings,
			"prompt requests JSON-only output; use EnableStructuredOutput with a ResponseSchema for deterministic JSON instead of prompt-based instruction")
	}

	// 3.x specific: instructing model to "think step by step" is redundant.
	if modelID != "" {
		model, err := GetModel(modelID)
		if err == nil && model.SupportsThinking {
			if strings.Contains(lower, "think step by step") || strings.Contains(lower, "chain of thought") {
				warnings = append(warnings,
					"model supports native thinking mode; 'think step by step' is redundant and may waste output tokens -- use ThinkingConfig instead")
			}
		}
	}

	return warnings
}
