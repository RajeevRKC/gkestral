package gemini

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

// SchemaFromStruct generates a Gemini-compatible ResponseSchema from a Go struct.
// Uses reflect to inspect struct tags and types. Supports nested structs,
// slices, and basic types (string, int, float64, bool).
// Handles self-referential structs by tracking visited types.
//
// Struct fields must have json tags. Fields with "required" in their gemini tag
// are marked as required: `gemini:"required"`
func SchemaFromStruct(v any) (*ResponseSchema, error) {
	t := reflect.TypeOf(v)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("expected struct, got %s", t.Kind())
	}
	visited := make(map[reflect.Type]bool)
	return structToSchema(t, visited)
}

// structToSchema converts a reflect.Type (struct) to a ResponseSchema.
// visited tracks types already being processed to prevent infinite recursion.
func structToSchema(t reflect.Type, visited map[reflect.Type]bool) (*ResponseSchema, error) {
	if visited[t] {
		// Self-referential type -- return opaque object to break cycle.
		return &ResponseSchema{Type: "object"}, nil
	}
	visited[t] = true
	defer delete(visited, t)

	schema := &ResponseSchema{
		Type:       "object",
		Properties: make(map[string]*ResponseSchema),
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}

		jsonTag := field.Tag.Get("json")
		if jsonTag == "" || jsonTag == "-" {
			continue
		}
		name := jsonTag
		if idx := strings.IndexByte(jsonTag, ','); idx != -1 {
			name = jsonTag[:idx]
		}
		if name == "" {
			continue
		}

		fieldSchema, err := typeToSchema(field.Type, visited)
		if err != nil {
			return nil, fmt.Errorf("field %s: %w", field.Name, err)
		}

		// Check for description in gemini tag.
		geminiTag := field.Tag.Get("gemini")
		if geminiTag != "" {
			if geminiTag == "required" {
				schema.Required = append(schema.Required, name)
			} else {
				fieldSchema.Description = geminiTag
			}
		}
		// Check for description tag.
		descTag := field.Tag.Get("description")
		if descTag != "" {
			fieldSchema.Description = descTag
		}

		schema.Properties[name] = fieldSchema
	}

	return schema, nil
}

// typeToSchema maps a Go type to a Gemini ResponseSchema type.
func typeToSchema(t reflect.Type, visited map[reflect.Type]bool) (*ResponseSchema, error) {
	// Dereference pointer.
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	switch t.Kind() {
	case reflect.String:
		return &ResponseSchema{Type: "string"}, nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return &ResponseSchema{Type: "integer"}, nil
	case reflect.Float32, reflect.Float64:
		return &ResponseSchema{Type: "number"}, nil
	case reflect.Bool:
		return &ResponseSchema{Type: "boolean"}, nil
	case reflect.Slice:
		itemSchema, err := typeToSchema(t.Elem(), visited)
		if err != nil {
			return nil, err
		}
		return &ResponseSchema{Type: "array", Items: itemSchema}, nil
	case reflect.Struct:
		return structToSchema(t, visited)
	case reflect.Map:
		return &ResponseSchema{Type: "object"}, nil
	default:
		return &ResponseSchema{Type: "string"}, nil
	}
}

// EnableStructuredOutput configures a request for deterministic JSON output.
// The model will respond with valid JSON matching the given schema.
func EnableStructuredOutput(request *GenerateContentRequest, schema *ResponseSchema) {
	if request.GenerationConfig == nil {
		request.GenerationConfig = &GenerationConfig{}
	}
	request.GenerationConfig.ResponseMIMEType = "application/json"
	request.GenerationConfig.ResponseSchema = schema
}

// ParseStructuredResponse parses a JSON response from structured output mode
// into the given target struct.
func ParseStructuredResponse(resp *GenerateContentResponse, target any) error {
	if target == nil {
		return fmt.Errorf("target must not be nil")
	}
	if resp == nil || len(resp.Candidates) == 0 {
		return fmt.Errorf("empty response")
	}

	content := resp.Candidates[0].Content
	if content == nil || len(content.Parts) == 0 {
		return fmt.Errorf("no content in response")
	}

	// Structured output returns JSON in the text part.
	text := content.Parts[0].Text
	if text == "" {
		return fmt.Errorf("empty text in response")
	}

	if err := json.Unmarshal([]byte(text), target); err != nil {
		return fmt.Errorf("parse structured output: %w", err)
	}
	return nil
}
