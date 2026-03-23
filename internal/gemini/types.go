// Package gemini provides a Go client library for the Gemini REST API.
// It implements raw REST + SSE streaming, context caching, thought signatures,
// function calling, search grounding, and model routing.
package gemini

import (
	"encoding/json"
	"fmt"
)

// EventType classifies the type of a provider event received during streaming.
type EventType int

const (
	// EventText indicates a text content chunk from the model.
	EventText EventType = iota
	// EventToolCall indicates the model is requesting a function call.
	EventToolCall
	// EventThoughtSignature indicates a thought/reasoning part from a 3.x model.
	EventThoughtSignature
	// EventError indicates an error occurred during streaming.
	EventError
	// EventDone indicates the stream has completed successfully.
	EventDone
)

// String returns the string representation of an EventType.
func (e EventType) String() string {
	switch e {
	case EventText:
		return "text"
	case EventToolCall:
		return "tool_call"
	case EventThoughtSignature:
		return "thought"
	case EventError:
		return "error"
	case EventDone:
		return "done"
	default:
		return fmt.Sprintf("unknown(%d)", int(e))
	}
}

// ProviderEvent represents a single event from a streaming Gemini response.
// Consumers read these from a channel returned by StreamResponse.
type ProviderEvent struct {
	Type    EventType     `json:"type"`
	Text    string        `json:"text,omitempty"`
	ToolCall *ToolCallData `json:"tool_call,omitempty"`
	Thought *ThoughtPart  `json:"thought,omitempty"`
	Error   error         `json:"error,omitempty"`
	Usage   *UsageMetadata `json:"usage,omitempty"`
}

// Message represents a single turn in a Gemini conversation.
type Message struct {
	Role  string `json:"role"` // "user", "model", or "function"
	Parts []Part `json:"parts"`
}

// Part represents a content part within a message. Only one field should be
// set at a time, matching the Gemini API's polymorphic part structure.
type Part struct {
	Text             string            `json:"text,omitempty"`
	InlineData       *InlineData       `json:"inlineData,omitempty"`
	FunctionCall     *FunctionCall     `json:"functionCall,omitempty"`
	FunctionResponse *FunctionResponse `json:"functionResponse,omitempty"`
	Thought          bool              `json:"thought,omitempty"`
}

// InlineData represents inline binary data (images, audio, etc.) in a message.
type InlineData struct {
	MIMEType string `json:"mimeType"`
	Data     string `json:"data"` // base64-encoded
}

// FunctionCall represents a function call request from the model.
type FunctionCall struct {
	Name string         `json:"name"`
	Args map[string]any `json:"args,omitempty"`
	ID   string         `json:"id,omitempty"`
}

// FunctionResponse represents the result of a function call sent back to the model.
type FunctionResponse struct {
	Name     string         `json:"name"`
	Response map[string]any `json:"response"`
	ID       string         `json:"id,omitempty"`
}

// FunctionDeclaration describes a function that the model can call.
// Parameters follow a subset of the OpenAPI Schema Object format.
type FunctionDeclaration struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters,omitempty"`
}

// ToolCallData represents a parsed tool call for dispatch.
type ToolCallData struct {
	ID           string         `json:"id"`
	FunctionName string         `json:"functionName"`
	Arguments    map[string]any `json:"arguments,omitempty"`
}

// ToolConfig configures function calling behaviour for a request.
type ToolConfig struct {
	FunctionCallingConfig *FunctionCallingConfig `json:"functionCallingConfig,omitempty"`
}

// FunctionCallingConfig controls how the model uses function declarations.
type FunctionCallingConfig struct {
	Mode             string   `json:"mode"`                       // AUTO, ANY, NONE
	AllowedFunctions []string `json:"allowedFunctionNames,omitempty"`
}

// ThoughtPart represents a reasoning/thinking part from a Gemini 3.x model.
type ThoughtPart struct {
	Text    string `json:"text"`
	Thought bool   `json:"thought"` // Must be true for valid thought parts
}

// UsageMetadata contains token usage information from a Gemini response.
type UsageMetadata struct {
	PromptTokenCount        int `json:"promptTokenCount"`
	CandidatesTokenCount    int `json:"candidatesTokenCount"`
	CachedContentTokenCount int `json:"cachedContentTokenCount,omitempty"`
	ThinkingTokenCount      int `json:"thoughtsTokenCount,omitempty"`
	TotalTokenCount         int `json:"totalTokenCount"`
}

// SafetySetting configures a safety filter for a specific harm category.
type SafetySetting struct {
	Category  string `json:"category"`
	Threshold string `json:"threshold"`
}

// GenerationConfig controls the model's output generation parameters.
type GenerationConfig struct {
	Temperature      *float64          `json:"temperature,omitempty"`
	TopP             *float64          `json:"topP,omitempty"`
	TopK             *int              `json:"topK,omitempty"`
	MaxOutputTokens  *int              `json:"maxOutputTokens,omitempty"`
	ResponseMIMEType string            `json:"responseMimeType,omitempty"`
	ResponseSchema   *ResponseSchema   `json:"responseSchema,omitempty"`
	ThinkingConfig   *ThinkingConfig   `json:"thinkingConfig,omitempty"`
}

// ThinkingConfig controls the model's internal reasoning behaviour.
type ThinkingConfig struct {
	ThinkingBudget int    `json:"thinkingBudget,omitempty"`
	ThinkingLevel  string `json:"thinkingLevel,omitempty"` // off, low, medium, high
}

// ResponseSchema defines the expected JSON schema for structured output responses.
// Mirrors a subset of the OpenAPI Schema Object used by the Gemini API.
type ResponseSchema struct {
	Type        string                    `json:"type"`
	Properties  map[string]*ResponseSchema `json:"properties,omitempty"`
	Required    []string                  `json:"required,omitempty"`
	Enum        []string                  `json:"enum,omitempty"`
	Items       *ResponseSchema           `json:"items,omitempty"`
	Description string                    `json:"description,omitempty"`
}

// ContentFilter holds information about a safety-blocked response.
type ContentFilter struct {
	Blocked bool     `json:"blocked"`
	Reasons []string `json:"reasons,omitempty"`
}

// CostEstimate represents the estimated cost for a Gemini API call.
type CostEstimate struct {
	InputCost       float64 `json:"inputCost"`
	OutputCost      float64 `json:"outputCost"`
	CachedInputCost float64 `json:"cachedInputCost"`
	TotalCost       float64 `json:"totalCost"`
	Savings         float64 `json:"savings"` // Savings from caching
}

// ---------- Gemini API Request/Response structures ----------

// GenerateContentRequest is the request body for the generateContent endpoint.
type GenerateContentRequest struct {
	Contents          []Message         `json:"contents"`
	SystemInstruction *Part             `json:"systemInstruction,omitempty"`
	Tools             []Tool            `json:"tools,omitempty"`
	ToolConfig        *ToolConfig       `json:"toolConfig,omitempty"`
	GenerationConfig  *GenerationConfig `json:"generationConfig,omitempty"`
	SafetySettings    []SafetySetting   `json:"safetySettings,omitempty"`
	CachedContent     string            `json:"cachedContent,omitempty"`
}

// Tool represents a tool declaration in a Gemini API request.
type Tool struct {
	FunctionDeclarations []FunctionDeclaration `json:"functionDeclarations,omitempty"`
	GoogleSearch         *GoogleSearch         `json:"googleSearch,omitempty"`
}

// GoogleSearch enables Google Search grounding in a Gemini API request.
type GoogleSearch struct {
	DynamicRetrievalConfig *DynamicRetrievalConfig `json:"dynamicRetrievalConfig,omitempty"`
}

// DynamicRetrievalConfig configures dynamic retrieval for search grounding.
type DynamicRetrievalConfig struct {
	Mode                               string `json:"mode,omitempty"` // VALIDATED for 3.x
	DynamicThreshold                   *float64 `json:"dynamicThreshold,omitempty"`
}

// GenerateContentResponse is the response from the generateContent endpoint.
type GenerateContentResponse struct {
	Candidates     []Candidate    `json:"candidates"`
	UsageMetadata  *UsageMetadata `json:"usageMetadata,omitempty"`
	PromptFeedback *PromptFeedback `json:"promptFeedback,omitempty"`
}

// Candidate represents a single response candidate from the model.
type Candidate struct {
	Content          *CandidateContent  `json:"content,omitempty"`
	FinishReason     string             `json:"finishReason,omitempty"`
	SafetyRatings    []SafetyRating     `json:"safetyRatings,omitempty"`
	GroundingMetadata *GroundingMetadata `json:"groundingMetadata,omitempty"`
}

// CandidateContent is the content structure within a Candidate.
type CandidateContent struct {
	Role  string `json:"role"`
	Parts []Part `json:"parts"`
}

// SafetyRating provides a safety assessment for a particular harm category.
type SafetyRating struct {
	Category    string `json:"category"`
	Probability string `json:"probability"`
	Blocked     bool   `json:"blocked"`
}

// PromptFeedback provides feedback about the prompt's safety assessment.
type PromptFeedback struct {
	BlockReason   string         `json:"blockReason,omitempty"`
	SafetyRatings []SafetyRating `json:"safetyRatings,omitempty"`
}

// GroundingMetadata contains metadata about Google Search grounding.
type GroundingMetadata struct {
	SearchEntryPoint  *SearchEntryPoint  `json:"searchEntryPoint,omitempty"`
	GroundingChunks   []GroundingChunk   `json:"groundingChunks,omitempty"`
	GroundingSupports []GroundingSupport `json:"groundingSupports,omitempty"`
	WebSearchQueries  []string           `json:"webSearchQueries,omitempty"`
}

// SearchEntryPoint represents the rendered search entry point.
type SearchEntryPoint struct {
	RenderedContent string `json:"renderedContent,omitempty"`
	SDKBlob         json.RawMessage `json:"sdkBlob,omitempty"`
}

// GroundingChunk represents a web source used for grounding.
type GroundingChunk struct {
	Web *WebChunk `json:"web,omitempty"`
}

// WebChunk contains the URI and title of a grounding web source.
type WebChunk struct {
	URI   string `json:"uri"`
	Title string `json:"title"`
}

// GroundingSupport represents a segment of text supported by grounding sources.
type GroundingSupport struct {
	Segment              *Segment  `json:"segment,omitempty"`
	GroundingChunkIndices []int    `json:"groundingChunkIndices,omitempty"`
	ConfidenceScores     []float64 `json:"confidenceScores,omitempty"`
}

// Segment identifies a portion of the response text.
type Segment struct {
	StartIndex int    `json:"startIndex"`
	EndIndex   int    `json:"endIndex"`
	Text       string `json:"text"`
}

// Citation represents a structured citation extracted from grounding metadata.
type Citation struct {
	URL           string  `json:"url"`
	Title         string  `json:"title"`
	Confidence    float64 `json:"confidence"`
	SupportedText string  `json:"supportedText"`
}

// CountTokensRequest is the request body for the countTokens endpoint.
type CountTokensRequest struct {
	Contents         []Message       `json:"contents"`
	GenerateContentRequest *GenerateContentRequest `json:"generateContentRequest,omitempty"`
}

// CountTokensResponse is the response from the countTokens endpoint.
type CountTokensResponse struct {
	TotalTokens int `json:"totalTokens"`
}

// CachedContentRequest is the request body for creating cached content.
type CachedContentRequest struct {
	Model             string                `json:"model"`
	Contents          []Message             `json:"contents,omitempty"`
	SystemInstruction *Part                 `json:"systemInstruction,omitempty"`
	Tools             []Tool                `json:"tools,omitempty"`
	DisplayName       string                `json:"displayName,omitempty"`
	TTL               string                `json:"ttl,omitempty"` // e.g., "3600s"
}

// CachedContentResponse is the response from the cached content endpoints.
type CachedContentResponse struct {
	Name             string         `json:"name"`
	Model            string         `json:"model"`
	DisplayName      string         `json:"displayName"`
	CreateTime       string         `json:"createTime"`
	UpdateTime       string         `json:"updateTime"`
	ExpireTime       string         `json:"expireTime"`
	UsageMetadata    *UsageMetadata `json:"usageMetadata,omitempty"`
}
