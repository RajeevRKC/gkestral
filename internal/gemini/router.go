package gemini

import (
	"strings"
)

// TaskClass classifies the type of work to determine model selection.
type TaskClass int

const (
	// TaskDeepReasoning requires deep analytical thinking.
	TaskDeepReasoning TaskClass = iota
	// TaskFastExtraction needs quick data extraction or transformation.
	TaskFastExtraction
	// TaskImageGeneration involves creating images.
	TaskImageGeneration
	// TaskCodeGeneration involves writing, debugging, or reviewing code.
	TaskCodeGeneration
	// TaskResearch requires factual search and grounding.
	TaskResearch
	// TaskConversation is general dialogue -- cheapest model.
	TaskConversation
	// TaskLargeContext works with very large input (>100K tokens).
	TaskLargeContext
)

// String returns the string representation of a TaskClass.
func (tc TaskClass) String() string {
	switch tc {
	case TaskDeepReasoning:
		return "deep_reasoning"
	case TaskFastExtraction:
		return "fast_extraction"
	case TaskImageGeneration:
		return "image_generation"
	case TaskCodeGeneration:
		return "code_generation"
	case TaskResearch:
		return "research"
	case TaskConversation:
		return "conversation"
	case TaskLargeContext:
		return "large_context"
	default:
		return "unknown"
	}
}

// RoutingRule maps a task class to a preferred model and configuration.
type RoutingRule struct {
	TaskClass         TaskClass
	PreferredModel    string
	FallbackModels    []string
	RequiresThinking  bool
	RequiresGrounding bool
	ThinkingLevel     string // For models that support thinking
}

// Router selects the optimal model for a given task using a configurable
// routing table.
type Router struct {
	rules map[TaskClass]RoutingRule
}

// NewRouter creates a Router with the default routing table.
func NewRouter() *Router {
	return &Router{
		rules: defaultRoutingRules(),
	}
}

// NewCustomRouter creates a Router with a custom routing table.
func NewCustomRouter(rules []RoutingRule) *Router {
	r := &Router{
		rules: make(map[TaskClass]RoutingRule, len(rules)),
	}
	for _, rule := range rules {
		r.rules[rule.TaskClass] = rule
	}
	return r
}

// RouteModel returns the ModelConfig and any client options for the given
// task class. Falls back to conversation defaults if the task class is
// not in the routing table.
func (r *Router) RouteModel(taskClass TaskClass) (ModelConfig, *RoutingRule, error) {
	rule, ok := r.rules[taskClass]
	if !ok {
		// Fall back to conversation.
		rule = r.rules[TaskConversation]
	}

	model, err := GetModel(rule.PreferredModel)
	if err != nil {
		// Try fallback models.
		for _, fb := range rule.FallbackModels {
			model, err = GetModel(fb)
			if err == nil {
				return model, &rule, nil
			}
		}
		return ModelConfig{}, nil, err
	}
	return model, &rule, nil
}

// GetRule returns the routing rule for a task class, or nil if not found.
func (r *Router) GetRule(taskClass TaskClass) *RoutingRule {
	rule, ok := r.rules[taskClass]
	if !ok {
		return nil
	}
	return &rule
}

// SetRule adds or updates a routing rule in the table.
func (r *Router) SetRule(rule RoutingRule) {
	r.rules[rule.TaskClass] = rule
}

// defaultRoutingRules returns the built-in routing table optimised for
// Gkestral's use cases.
func defaultRoutingRules() map[TaskClass]RoutingRule {
	return map[TaskClass]RoutingRule{
		TaskDeepReasoning: {
			TaskClass:        TaskDeepReasoning,
			PreferredModel:   "gemini-3.1-pro-preview",
			FallbackModels:   []string{"gemini-2.5-pro"},
			RequiresThinking: true,
			ThinkingLevel:    ThinkingLevelHigh,
		},
		TaskFastExtraction: {
			TaskClass:      TaskFastExtraction,
			PreferredModel: "gemini-3.1-flash",
			FallbackModels: []string{"gemini-2.5-flash"},
		},
		TaskImageGeneration: {
			TaskClass:      TaskImageGeneration,
			PreferredModel: "gemini-3.1-flash-image-preview",
			FallbackModels: nil, // No fallback for image gen
		},
		TaskCodeGeneration: {
			TaskClass:        TaskCodeGeneration,
			PreferredModel:   "gemini-3.1-flash",
			FallbackModels:   []string{"gemini-2.5-flash"},
			RequiresThinking: true,
			ThinkingLevel:    ThinkingLevelMedium,
		},
		TaskResearch: {
			TaskClass:         TaskResearch,
			PreferredModel:    "gemini-3.1-pro-preview",
			FallbackModels:    []string{"gemini-2.5-pro"},
			RequiresGrounding: true,
		},
		TaskConversation: {
			TaskClass:      TaskConversation,
			PreferredModel: "gemini-2.5-flash",
			FallbackModels: nil,
		},
		TaskLargeContext: {
			TaskClass:      TaskLargeContext,
			PreferredModel: "gemini-2.5-pro",
			FallbackModels: []string{"gemini-2.5-flash"},
		},
	}
}

// ClassifyTask applies heuristic classification to determine the task type
// based on the prompt text and request characteristics. This is intentionally
// simple (no LLM call) -- it uses keyword matching and structural signals.
func ClassifyTask(prompt string, hasTools bool, hasImages bool) TaskClass {
	if hasImages {
		return TaskImageGeneration
	}

	lower := strings.ToLower(prompt)

	// Deep reasoning: long prompts with analytical keywords.
	if len(prompt) > 2000 && containsAny(lower, deepReasoningKeywords) {
		return TaskDeepReasoning
	}

	// Code generation: programming-related keywords.
	if containsAny(lower, codeKeywords) {
		return TaskCodeGeneration
	}

	// Research: factual questions with search intent.
	if containsAny(lower, researchKeywords) {
		return TaskResearch
	}

	// Fast extraction: short prompts asking for specific data.
	if len(prompt) < 500 && containsAny(lower, extractionKeywords) {
		return TaskFastExtraction
	}

	// Deep reasoning (secondary): analysis keywords without length filter.
	if containsAny(lower, deepReasoningKeywords) {
		return TaskDeepReasoning
	}

	// Large context: very long prompts.
	if len(prompt) > 50000 {
		return TaskLargeContext
	}

	return TaskConversation
}

// Keyword lists for classification heuristics.
var deepReasoningKeywords = []string{
	"analyze", "analyse", "architecture", "compare", "evaluate",
	"trade-off", "tradeoff", "design", "strategy", "systematic",
	"comprehensive", "in-depth", "deep dive", "reasoning",
	"implications", "consequences", "root cause",
}

var codeKeywords = []string{
	"function", "variable", "debug", "error", "compile",
	"refactor", "implement", "class", "struct", "interface",
	"api", "endpoint", "algorithm", "code", "program",
	"bug", "fix", "test", "unittest", "benchmark",
	"golang", "python", "javascript", "typescript", "rust",
}

var researchKeywords = []string{
	"what is", "who is", "when did", "how does", "why does",
	"latest", "current", "recent", "statistics", "data",
	"population", "capital", "founded", "history of",
	"explain how", "tell me about",
}

var extractionKeywords = []string{
	"extract", "parse", "convert", "transform", "summarize",
	"summarise", "list", "enumerate", "count", "format",
	"translate", "classify", "categorize", "categorise",
}

// containsAny returns true if s contains any of the given substrings.
func containsAny(s string, substrs []string) bool {
	for _, sub := range substrs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}
