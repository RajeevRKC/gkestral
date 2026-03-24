package gemini

import (
	"encoding/json"
	"testing"
)

type TestPerson struct {
	Name  string `json:"name" gemini:"required"`
	Age   int    `json:"age" description:"Person's age in years"`
	Email string `json:"email,omitempty"`
}

type TestReport struct {
	Title    string       `json:"title" gemini:"required"`
	Sections []TestSection `json:"sections"`
	Score    float64      `json:"score"`
	Valid    bool         `json:"valid"`
}

type TestSection struct {
	Heading string `json:"heading"`
	Content string `json:"content"`
}

func TestSchemaFromStruct_Simple(t *testing.T) {
	schema, err := SchemaFromStruct(TestPerson{})
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	if schema.Type != "object" {
		t.Errorf("type: want 'object', got %q", schema.Type)
	}

	if len(schema.Properties) != 3 {
		t.Fatalf("properties: want 3, got %d", len(schema.Properties))
	}

	name := schema.Properties["name"]
	if name == nil || name.Type != "string" {
		t.Error("name property missing or wrong type")
	}

	age := schema.Properties["age"]
	if age == nil || age.Type != "integer" {
		t.Error("age property missing or wrong type")
	}
	if age.Description != "Person's age in years" {
		t.Errorf("age description: got %q", age.Description)
	}

	// Check required fields.
	if len(schema.Required) != 1 || schema.Required[0] != "name" {
		t.Errorf("required: want [name], got %v", schema.Required)
	}
}

func TestSchemaFromStruct_Nested(t *testing.T) {
	schema, err := SchemaFromStruct(TestReport{})
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	sections := schema.Properties["sections"]
	if sections == nil || sections.Type != "array" {
		t.Fatal("sections: want array type")
	}
	if sections.Items == nil || sections.Items.Type != "object" {
		t.Error("sections.items: want object type")
	}
	if len(sections.Items.Properties) != 2 {
		t.Errorf("sections.items properties: want 2, got %d", len(sections.Items.Properties))
	}

	score := schema.Properties["score"]
	if score == nil || score.Type != "number" {
		t.Error("score: want number type")
	}

	valid := schema.Properties["valid"]
	if valid == nil || valid.Type != "boolean" {
		t.Error("valid: want boolean type")
	}
}

func TestSchemaFromStruct_Pointer(t *testing.T) {
	schema, err := SchemaFromStruct(&TestPerson{})
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if schema.Type != "object" {
		t.Error("pointer to struct should work")
	}
}

func TestSchemaFromStruct_NonStruct(t *testing.T) {
	_, err := SchemaFromStruct("not a struct")
	if err == nil {
		t.Error("expected error for non-struct input")
	}
}

func TestSchemaJSON(t *testing.T) {
	schema, err := SchemaFromStruct(TestPerson{})
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	data, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var parsed ResponseSchema
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if parsed.Type != "object" || len(parsed.Properties) != 3 {
		t.Error("round-trip failed")
	}
}

func TestEnableStructuredOutput(t *testing.T) {
	schema := &ResponseSchema{
		Type: "object",
		Properties: map[string]*ResponseSchema{
			"result": {Type: "string"},
		},
	}

	req := &GenerateContentRequest{}
	EnableStructuredOutput(req, schema)

	if req.GenerationConfig == nil {
		t.Fatal("GenerationConfig should be set")
	}
	if req.GenerationConfig.ResponseMIMEType != "application/json" {
		t.Errorf("MIME type: want 'application/json', got %q", req.GenerationConfig.ResponseMIMEType)
	}
	if req.GenerationConfig.ResponseSchema == nil {
		t.Fatal("ResponseSchema should be set")
	}
}

func TestParseStructuredResponse(t *testing.T) {
	resp := &GenerateContentResponse{
		Candidates: []Candidate{{
			Content: &CandidateContent{
				Role:  "model",
				Parts: []Part{{Text: `{"name":"Alice","age":30,"email":"alice@example.com"}`}},
			},
		}},
	}

	var person TestPerson
	err := ParseStructuredResponse(resp, &person)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if person.Name != "Alice" {
		t.Errorf("name: want 'Alice', got %q", person.Name)
	}
	if person.Age != 30 {
		t.Errorf("age: want 30, got %d", person.Age)
	}
}

func TestParseStructuredResponse_InvalidJSON(t *testing.T) {
	resp := &GenerateContentResponse{
		Candidates: []Candidate{{
			Content: &CandidateContent{
				Role:  "model",
				Parts: []Part{{Text: `not valid json`}},
			},
		}},
	}

	var person TestPerson
	err := ParseStructuredResponse(resp, &person)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestParseStructuredResponse_Empty(t *testing.T) {
	if err := ParseStructuredResponse(nil, &TestPerson{}); err == nil {
		t.Error("expected error for nil response")
	}
	if err := ParseStructuredResponse(&GenerateContentResponse{}, &TestPerson{}); err == nil {
		t.Error("expected error for empty response")
	}
}
