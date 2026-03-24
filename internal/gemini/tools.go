package gemini

import (
	"context"
	"fmt"
	"sync"
)

// ToolExecutor is a function that executes a single tool call and returns
// the result as a map.
type ToolExecutor func(ctx context.Context, call ToolCallData) (map[string]any, error)

// ToolRegistry holds registered tool declarations and their executors.
type ToolRegistry struct {
	mu          sync.RWMutex
	declarations map[string]FunctionDeclaration
	executors    map[string]ToolExecutor
}

// NewToolRegistry creates an empty ToolRegistry.
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		declarations: make(map[string]FunctionDeclaration),
		executors:    make(map[string]ToolExecutor),
	}
}

// RegisterTool adds a tool declaration and optional executor to the registry.
// If executor is nil the tool is declared but dispatch will fail with an error.
func (r *ToolRegistry) RegisterTool(decl FunctionDeclaration, executor ToolExecutor) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.declarations[decl.Name] = decl
	if executor != nil {
		r.executors[decl.Name] = executor
	}
}

// RegisterTools adds multiple tool declarations at once without executors.
// Use RegisterTool for declarations with executors.
func (r *ToolRegistry) RegisterTools(decls []FunctionDeclaration) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, d := range decls {
		r.declarations[d.Name] = d
	}
}

// Declarations returns all registered declarations as a slice suitable for
// inclusion in a Tool struct's FunctionDeclarations field.
func (r *ToolRegistry) Declarations() []FunctionDeclaration {
	r.mu.RLock()
	defer r.mu.RUnlock()
	decls := make([]FunctionDeclaration, 0, len(r.declarations))
	for _, d := range r.declarations {
		decls = append(decls, d)
	}
	return decls
}

// ParseToolCalls extracts function call requests from a Gemini response.
// Each FunctionCall in the response's Parts becomes a ToolCallData with
// its ID preserved for parallel call matching.
func ParseToolCalls(resp *GenerateContentResponse) []ToolCallData {
	if resp == nil || len(resp.Candidates) == 0 {
		return nil
	}

	content := resp.Candidates[0].Content
	if content == nil {
		return nil
	}

	var calls []ToolCallData
	for _, part := range content.Parts {
		if part.FunctionCall != nil {
			calls = append(calls, ToolCallData{
				ID:           part.FunctionCall.ID,
				FunctionName: part.FunctionCall.Name,
				Arguments:    part.FunctionCall.Args,
			})
		}
	}
	return calls
}

// BuildToolResponse constructs function response Parts matched by ID for
// sending back to the model. Each result is paired with its originating
// call via the ID field.
func BuildToolResponse(calls []ToolCallData, results []map[string]any) []Part {
	parts := make([]Part, 0, len(calls))
	for i, call := range calls {
		var result map[string]any
		if i < len(results) {
			result = results[i]
		}
		if result == nil {
			result = map[string]any{"error": "no result"}
		}
		parts = append(parts, Part{
			FunctionResponse: &FunctionResponse{
				Name:     call.FunctionName,
				Response: result,
				ID:       call.ID,
			},
		})
	}
	return parts
}

// DispatchParallel executes multiple tool calls concurrently and returns
// results in the same order as the input calls. Each result is matched
// to its originating call by index position. Errors from individual
// calls are captured in the result map under an "error" key.
func DispatchParallel(ctx context.Context, calls []ToolCallData, executor ToolExecutor) []map[string]any {
	if len(calls) == 0 {
		return nil
	}

	results := make([]map[string]any, len(calls))
	var wg sync.WaitGroup
	wg.Add(len(calls))

	for i, call := range calls {
		go func(idx int, c ToolCallData) {
			defer wg.Done()

			result, err := executor(ctx, c)
			if err != nil {
				results[idx] = map[string]any{"error": err.Error()}
				return
			}
			results[idx] = result
		}(i, call)
	}

	wg.Wait()
	return results
}

// DispatchSequential executes tool calls one at a time in order.
// Useful when tools have side effects or ordering dependencies.
// Respects context cancellation: remaining calls are skipped if ctx is done.
func DispatchSequential(ctx context.Context, calls []ToolCallData, executor ToolExecutor) []map[string]any {
	results := make([]map[string]any, len(calls))
	for i, call := range calls {
		// Check context before each call.
		select {
		case <-ctx.Done():
			results[i] = map[string]any{"error": ctx.Err().Error()}
			// Fill remaining results with cancellation error.
			for j := i + 1; j < len(calls); j++ {
				results[j] = map[string]any{"error": ctx.Err().Error()}
			}
			return results
		default:
		}

		result, err := executor(ctx, call)
		if err != nil {
			results[i] = map[string]any{"error": err.Error()}
			continue
		}
		results[i] = result
	}
	return results
}

// DispatchFromRegistry uses the ToolRegistry's registered executors to
// dispatch calls in parallel, looking up each call by function name.
func (r *ToolRegistry) DispatchFromRegistry(ctx context.Context, calls []ToolCallData) []map[string]any {
	return DispatchParallel(ctx, calls, func(ctx context.Context, call ToolCallData) (map[string]any, error) {
		r.mu.RLock()
		executor, ok := r.executors[call.FunctionName]
		r.mu.RUnlock()

		if !ok {
			return nil, fmt.Errorf("no executor registered for tool %q", call.FunctionName)
		}
		return executor(ctx, call)
	})
}

// DeclareTool is a convenience helper for building a FunctionDeclaration
// from a name, description, and parameter map.
func DeclareTool(name, description string, params map[string]any) FunctionDeclaration {
	return FunctionDeclaration{
		Name:        name,
		Description: description,
		Parameters:  params,
	}
}

// ---------- JSON Schema helpers ----------

// SchemaString creates a string property schema.
func SchemaString(description string) map[string]any {
	return map[string]any{
		"type":        "string",
		"description": description,
	}
}

// SchemaInt creates an integer property schema.
func SchemaInt(description string) map[string]any {
	return map[string]any{
		"type":        "integer",
		"description": description,
	}
}

// SchemaNumber creates a number (float) property schema.
func SchemaNumber(description string) map[string]any {
	return map[string]any{
		"type":        "number",
		"description": description,
	}
}

// SchemaBool creates a boolean property schema.
func SchemaBool(description string) map[string]any {
	return map[string]any{
		"type":        "boolean",
		"description": description,
	}
}

// SchemaArray creates an array property schema with the given item type.
func SchemaArray(description string, items map[string]any) map[string]any {
	return map[string]any{
		"type":        "array",
		"description": description,
		"items":       items,
	}
}

// SchemaObject creates an object property schema.
func SchemaObject(description string, properties map[string]any, required []string) map[string]any {
	s := map[string]any{
		"type":        "object",
		"description": description,
		"properties":  properties,
	}
	if len(required) > 0 {
		s["required"] = required
	}
	return s
}

// SchemaEnum creates a string enum property schema.
func SchemaEnum(description string, values []string) map[string]any {
	return map[string]any{
		"type":        "string",
		"description": description,
		"enum":        values,
	}
}

// BuildToolConfig creates a ToolConfig for controlling function calling mode.
// Mode: "AUTO" (model decides), "ANY" (force a call), "NONE" (disable calls).
func BuildToolConfig(mode string, allowedFunctions ...string) *ToolConfig {
	cfg := &ToolConfig{
		FunctionCallingConfig: &FunctionCallingConfig{
			Mode: mode,
		},
	}
	if len(allowedFunctions) > 0 {
		cfg.FunctionCallingConfig.AllowedFunctions = allowedFunctions
	}
	return cfg
}

// ApplyTools sets the function declarations and optional tool config on a
// GenerateContentRequest.
func ApplyTools(request *GenerateContentRequest, declarations []FunctionDeclaration, config *ToolConfig) {
	if len(declarations) > 0 {
		// Add to the first Tool entry (or create one).
		if len(request.Tools) == 0 {
			request.Tools = []Tool{{}}
		}
		request.Tools[0].FunctionDeclarations = declarations
	}
	if config != nil {
		request.ToolConfig = config
	}
}
