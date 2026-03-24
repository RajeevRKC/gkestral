package gemini

import (
	"context"
	"fmt"
	"sort"
	"sync/atomic"
	"testing"
)

func TestToolRegistry_RegisterAndDeclare(t *testing.T) {
	r := NewToolRegistry()
	r.RegisterTool(
		FunctionDeclaration{Name: "read_file", Description: "Read a file"},
		func(_ context.Context, call ToolCallData) (map[string]any, error) {
			return map[string]any{"content": "hello"}, nil
		},
	)
	r.RegisterTool(
		FunctionDeclaration{Name: "list_dir", Description: "List a directory"},
		nil,
	)

	decls := r.Declarations()
	if len(decls) != 2 {
		t.Fatalf("got %d declarations, want 2", len(decls))
	}

	// Sort for deterministic comparison.
	sort.Slice(decls, func(i, j int) bool { return decls[i].Name < decls[j].Name })
	if decls[0].Name != "list_dir" {
		t.Errorf("decls[0].Name = %q, want %q", decls[0].Name, "list_dir")
	}
	if decls[1].Name != "read_file" {
		t.Errorf("decls[1].Name = %q, want %q", decls[1].Name, "read_file")
	}
}

func TestToolRegistry_RegisterTools(t *testing.T) {
	r := NewToolRegistry()
	r.RegisterTools([]FunctionDeclaration{
		{Name: "tool_a", Description: "A"},
		{Name: "tool_b", Description: "B"},
		{Name: "tool_c", Description: "C"},
	})

	decls := r.Declarations()
	if len(decls) != 3 {
		t.Fatalf("got %d declarations, want 3", len(decls))
	}
}

func TestParseToolCalls_Single(t *testing.T) {
	resp := &GenerateContentResponse{
		Candidates: []Candidate{
			{
				Content: &CandidateContent{
					Role: "model",
					Parts: []Part{
						{
							FunctionCall: &FunctionCall{
								Name: "get_weather",
								Args: map[string]any{"city": "Riyadh"},
								ID:   "call_001",
							},
						},
					},
				},
			},
		},
	}

	calls := ParseToolCalls(resp)
	if len(calls) != 1 {
		t.Fatalf("got %d calls, want 1", len(calls))
	}
	if calls[0].FunctionName != "get_weather" {
		t.Errorf("FunctionName = %q, want %q", calls[0].FunctionName, "get_weather")
	}
	if calls[0].ID != "call_001" {
		t.Errorf("ID = %q, want %q", calls[0].ID, "call_001")
	}
	if calls[0].Arguments["city"] != "Riyadh" {
		t.Errorf("city = %v, want %q", calls[0].Arguments["city"], "Riyadh")
	}
}

func TestParseToolCalls_Parallel(t *testing.T) {
	resp := &GenerateContentResponse{
		Candidates: []Candidate{
			{
				Content: &CandidateContent{
					Role: "model",
					Parts: []Part{
						{
							FunctionCall: &FunctionCall{
								Name: "get_weather",
								Args: map[string]any{"city": "Riyadh"},
								ID:   "call_001",
							},
						},
						{
							FunctionCall: &FunctionCall{
								Name: "get_time",
								Args: map[string]any{"timezone": "AST"},
								ID:   "call_002",
							},
						},
						{
							FunctionCall: &FunctionCall{
								Name: "get_news",
								Args: map[string]any{"topic": "tech"},
								ID:   "call_003",
							},
						},
					},
				},
			},
		},
	}

	calls := ParseToolCalls(resp)
	if len(calls) != 3 {
		t.Fatalf("got %d calls, want 3", len(calls))
	}

	// Verify IDs are preserved.
	for i, wantID := range []string{"call_001", "call_002", "call_003"} {
		if calls[i].ID != wantID {
			t.Errorf("calls[%d].ID = %q, want %q", i, calls[i].ID, wantID)
		}
	}
}

func TestParseToolCalls_NilResponse(t *testing.T) {
	calls := ParseToolCalls(nil)
	if calls != nil {
		t.Errorf("expected nil for nil response, got %v", calls)
	}
}

func TestParseToolCalls_NoCalls(t *testing.T) {
	resp := &GenerateContentResponse{
		Candidates: []Candidate{
			{
				Content: &CandidateContent{
					Role:  "model",
					Parts: []Part{{Text: "no tools needed"}},
				},
			},
		},
	}

	calls := ParseToolCalls(resp)
	if len(calls) != 0 {
		t.Errorf("expected 0 calls, got %d", len(calls))
	}
}

func TestBuildToolResponse(t *testing.T) {
	calls := []ToolCallData{
		{ID: "call_001", FunctionName: "get_weather"},
		{ID: "call_002", FunctionName: "get_time"},
	}
	results := []map[string]any{
		{"temperature": 42, "unit": "celsius"},
		{"time": "14:30", "timezone": "AST"},
	}

	parts := BuildToolResponse(calls, results)
	if len(parts) != 2 {
		t.Fatalf("got %d parts, want 2", len(parts))
	}

	// Verify ID matching.
	if parts[0].FunctionResponse.ID != "call_001" {
		t.Errorf("parts[0].ID = %q, want %q", parts[0].FunctionResponse.ID, "call_001")
	}
	if parts[0].FunctionResponse.Name != "get_weather" {
		t.Errorf("parts[0].Name = %q, want %q", parts[0].FunctionResponse.Name, "get_weather")
	}
	if parts[1].FunctionResponse.ID != "call_002" {
		t.Errorf("parts[1].ID = %q, want %q", parts[1].FunctionResponse.ID, "call_002")
	}
}

func TestBuildToolResponse_MissingResults(t *testing.T) {
	calls := []ToolCallData{
		{ID: "call_001", FunctionName: "get_weather"},
		{ID: "call_002", FunctionName: "get_time"},
	}
	// Provide fewer results than calls.
	results := []map[string]any{
		{"temperature": 42},
	}

	parts := BuildToolResponse(calls, results)
	if len(parts) != 2 {
		t.Fatalf("got %d parts, want 2", len(parts))
	}

	// Second part should have error fallback.
	if parts[1].FunctionResponse.Response["error"] != "no result" {
		t.Errorf("expected fallback error for missing result, got %v", parts[1].FunctionResponse.Response)
	}
}

func TestDispatchParallel(t *testing.T) {
	calls := []ToolCallData{
		{ID: "call_001", FunctionName: "add", Arguments: map[string]any{"a": float64(1), "b": float64(2)}},
		{ID: "call_002", FunctionName: "multiply", Arguments: map[string]any{"a": float64(3), "b": float64(4)}},
		{ID: "call_003", FunctionName: "subtract", Arguments: map[string]any{"a": float64(10), "b": float64(3)}},
	}

	var concurrentCount atomic.Int32
	var maxConcurrent atomic.Int32

	executor := func(_ context.Context, call ToolCallData) (map[string]any, error) {
		current := concurrentCount.Add(1)
		// Track max concurrency observed.
		for {
			old := maxConcurrent.Load()
			if current <= old || maxConcurrent.CompareAndSwap(old, current) {
				break
			}
		}

		a, _ := call.Arguments["a"].(float64)
		b, _ := call.Arguments["b"].(float64)

		var result float64
		switch call.FunctionName {
		case "add":
			result = a + b
		case "multiply":
			result = a * b
		case "subtract":
			result = a - b
		}

		concurrentCount.Add(-1)
		return map[string]any{"result": result}, nil
	}

	results := DispatchParallel(context.Background(), calls, executor)
	if len(results) != 3 {
		t.Fatalf("got %d results, want 3", len(results))
	}

	// Verify results match expected values.
	expected := []float64{3, 12, 7}
	for i, want := range expected {
		got, ok := results[i]["result"].(float64)
		if !ok {
			t.Errorf("results[%d] = %v, expected float64", i, results[i])
			continue
		}
		if got != want {
			t.Errorf("results[%d] = %v, want %v", i, got, want)
		}
	}
}

func TestDispatchParallel_ErrorInOneCall(t *testing.T) {
	calls := []ToolCallData{
		{ID: "call_001", FunctionName: "good"},
		{ID: "call_002", FunctionName: "bad"},
	}

	executor := func(_ context.Context, call ToolCallData) (map[string]any, error) {
		if call.FunctionName == "bad" {
			return nil, fmt.Errorf("tool execution failed")
		}
		return map[string]any{"status": "ok"}, nil
	}

	results := DispatchParallel(context.Background(), calls, executor)
	if len(results) != 2 {
		t.Fatalf("got %d results, want 2", len(results))
	}

	// Good call should succeed.
	if results[0]["status"] != "ok" {
		t.Errorf("results[0] = %v, want status=ok", results[0])
	}

	// Bad call should have error.
	errMsg, ok := results[1]["error"].(string)
	if !ok || errMsg == "" {
		t.Errorf("results[1] should have error, got %v", results[1])
	}
}

func TestDispatchParallel_Empty(t *testing.T) {
	results := DispatchParallel(context.Background(), nil, func(_ context.Context, _ ToolCallData) (map[string]any, error) {
		return nil, nil
	})
	if results != nil {
		t.Errorf("expected nil for empty calls, got %v", results)
	}
}

func TestDispatchSequential(t *testing.T) {
	order := make([]string, 0)
	calls := []ToolCallData{
		{ID: "1", FunctionName: "first"},
		{ID: "2", FunctionName: "second"},
		{ID: "3", FunctionName: "third"},
	}

	executor := func(_ context.Context, call ToolCallData) (map[string]any, error) {
		order = append(order, call.FunctionName)
		return map[string]any{"name": call.FunctionName}, nil
	}

	results := DispatchSequential(context.Background(), calls, executor)
	if len(results) != 3 {
		t.Fatalf("got %d results, want 3", len(results))
	}

	// Verify sequential execution order.
	for i, want := range []string{"first", "second", "third"} {
		if order[i] != want {
			t.Errorf("order[%d] = %q, want %q", i, order[i], want)
		}
	}
}

func TestToolRegistry_DispatchFromRegistry(t *testing.T) {
	r := NewToolRegistry()
	r.RegisterTool(
		FunctionDeclaration{Name: "greet", Description: "Greet a person"},
		func(_ context.Context, call ToolCallData) (map[string]any, error) {
			name, _ := call.Arguments["name"].(string)
			return map[string]any{"greeting": "Hello, " + name + "!"}, nil
		},
	)

	calls := []ToolCallData{
		{ID: "1", FunctionName: "greet", Arguments: map[string]any{"name": "Commander"}},
	}

	results := r.DispatchFromRegistry(context.Background(), calls)
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if results[0]["greeting"] != "Hello, Commander!" {
		t.Errorf("greeting = %v, want %q", results[0]["greeting"], "Hello, Commander!")
	}
}

func TestToolRegistry_DispatchFromRegistry_UnknownTool(t *testing.T) {
	r := NewToolRegistry()
	calls := []ToolCallData{
		{ID: "1", FunctionName: "unknown_tool"},
	}

	results := r.DispatchFromRegistry(context.Background(), calls)
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if _, ok := results[0]["error"]; !ok {
		t.Error("expected error for unknown tool")
	}
}

func TestDeclareTool(t *testing.T) {
	params := SchemaObject("Parameters", map[string]any{
		"city": SchemaString("The city name"),
		"unit": SchemaEnum("Temperature unit", []string{"celsius", "fahrenheit"}),
	}, []string{"city"})

	decl := DeclareTool("get_weather", "Get current weather", params)
	if decl.Name != "get_weather" {
		t.Errorf("Name = %q, want %q", decl.Name, "get_weather")
	}
	if decl.Description != "Get current weather" {
		t.Errorf("Description = %q, want %q", decl.Description, "Get current weather")
	}
	if decl.Parameters == nil {
		t.Error("Parameters should not be nil")
	}
}

func TestSchemaHelpers(t *testing.T) {
	tests := []struct {
		name     string
		schema   map[string]any
		wantType string
	}{
		{"SchemaString", SchemaString("A string"), "string"},
		{"SchemaInt", SchemaInt("An integer"), "integer"},
		{"SchemaNumber", SchemaNumber("A number"), "number"},
		{"SchemaBool", SchemaBool("A boolean"), "boolean"},
		{"SchemaArray", SchemaArray("An array", SchemaString("item")), "array"},
		{"SchemaObject", SchemaObject("An object", nil, nil), "object"},
		{"SchemaEnum", SchemaEnum("An enum", []string{"a", "b"}), "string"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.schema["type"]; got != tt.wantType {
				t.Errorf("type = %v, want %q", got, tt.wantType)
			}
		})
	}
}

func TestSchemaEnum_Values(t *testing.T) {
	s := SchemaEnum("test", []string{"red", "green", "blue"})
	vals, ok := s["enum"].([]string)
	if !ok {
		t.Fatal("enum should be []string")
	}
	if len(vals) != 3 {
		t.Errorf("enum length = %d, want 3", len(vals))
	}
}

func TestBuildToolConfig(t *testing.T) {
	tests := []struct {
		name    string
		mode    string
		allowed []string
	}{
		{"auto mode", "AUTO", nil},
		{"any mode with filter", "ANY", []string{"get_weather"}},
		{"none mode", "NONE", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := BuildToolConfig(tt.mode, tt.allowed...)
			if cfg.FunctionCallingConfig.Mode != tt.mode {
				t.Errorf("mode = %q, want %q", cfg.FunctionCallingConfig.Mode, tt.mode)
			}
			if len(tt.allowed) > 0 && len(cfg.FunctionCallingConfig.AllowedFunctions) != len(tt.allowed) {
				t.Errorf("allowed functions = %d, want %d", len(cfg.FunctionCallingConfig.AllowedFunctions), len(tt.allowed))
			}
		})
	}
}

func TestApplyTools(t *testing.T) {
	req := &GenerateContentRequest{
		Contents: []Message{{Role: "user", Parts: []Part{{Text: "test"}}}},
	}

	decls := []FunctionDeclaration{
		{Name: "tool_a", Description: "A"},
		{Name: "tool_b", Description: "B"},
	}
	cfg := BuildToolConfig("AUTO")

	ApplyTools(req, decls, cfg)

	if len(req.Tools) != 1 {
		t.Fatalf("Tools length = %d, want 1", len(req.Tools))
	}
	if len(req.Tools[0].FunctionDeclarations) != 2 {
		t.Errorf("FunctionDeclarations = %d, want 2", len(req.Tools[0].FunctionDeclarations))
	}
	if req.ToolConfig == nil {
		t.Error("ToolConfig should not be nil")
	}
	if req.ToolConfig.FunctionCallingConfig.Mode != "AUTO" {
		t.Errorf("mode = %q, want %q", req.ToolConfig.FunctionCallingConfig.Mode, "AUTO")
	}
}
