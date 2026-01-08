package compilation

import (
	"encoding/json"
	"fmt"
)

type SchemaType string

const (
	SchemaNull    SchemaType = "null"
	SchemaBoolean SchemaType = "boolean"
	SchemaInteger SchemaType = "integer"
	SchemaNumber  SchemaType = "number"
	SchemaString  SchemaType = "string"
	SchemaArray   SchemaType = "array"
	SchemaObject  SchemaType = "object"
)

type Schema struct {
	Type SchemaType `json:"type" yaml:"type"`

	Properties map[string]SchemaOrRef `json:"properties,omitempty" yaml:"properties,omitempty"`
	Items      *SchemaOrRef           `json:"items,omitempty" yaml:"items,omitempty"`

	nullable bool // `json:"nullable,omitempty" yaml:"nullable,omitempty"`

	Default *any `json:"default,omitempty" yaml:"default,omitempty"`

	Required []string `json:"required,omitempty" yaml:"required,omitempty"`

	Format string `json:"format,omitempty" yaml:"format,omitempty"`

	UniqueItems bool `json:"uniqueItems,omitempty" yaml:"uniqueItems,omitempty"`

	Minimum *int `json:"minimum,omitempty" yaml:"minimum,omitempty"`
	Maximum *int `json:"maximum,omitempty" yaml:"maximum,omitempty"`

	MinLength *uint `json:"minLength,omitempty" yaml:"minLength,omitempty"`
	MaxLength *uint `json:"maxLength,omitempty" yaml:"maxLength,omitempty"`

	MinItems *uint `json:"minItems,omitempty" yaml:"minItems,omitempty"`
	MaxItems *uint `json:"maxItems,omitempty" yaml:"maxItems,omitempty"`

	Examples []any `json:"examples,omitempty" yaml:"examples,omitempty"`
}

type SchemaOrRef struct {
	value any
}

func (SchemaOrRef) IsEmpty() bool { return false }
func (SchemaOrRef) IsZero() bool  { return false }

func NewSchemaRef(ref string) SchemaOrRef {
	return SchemaOrRef{
		value: ref,
	}
}

func NewSchemaDef(schema Schema) SchemaOrRef {
	return SchemaOrRef{
		value: schema,
	}
}
func (t Schema) marshalYAML(nullable bool) (any, error) {
	if nullable {
		// produce: oneOf: [ {type: "null"}, <schema-without-nullable> ]
		nonNull := t
		nonNull.nullable = false // ensure the inner schema is not nullable

		return map[string]any{
			"oneOf": []any{
				map[string]string{"type": string(SchemaNull)},
				nonNull,
			},
		}, nil
	}

	// default: marshal normally as the Schema struct (nullable field is internal)
	return t, nil
}

func (t SchemaOrRef) MarshalYAML() (any, error) {
	if t.value == nil {
		return nil, nil
	}

	switch v := t.value.(type) {
	case string:
		return map[string]string{"$ref": v}, nil
	case Schema:
		return v.marshalYAML(v.nullable)
	default:
		return nil, fmt.Errorf("invalid SchemaOrRef value type: %T", v)
	}
}
func (t SchemaOrRef) MarshalJSON() ([]byte, error) {
	switch v := t.value.(type) {
	case string:
		// reference object: {"$ref": "..."}
		return json.Marshal(map[string]string{"$ref": v})
	case Schema:
		// if schema is nullable, produce: {"oneOf":[{"type":"null"}, <schema-without-nullable>]}
		if v.nullable {
			nonNull := v
			nonNull.nullable = false // avoid infinite recursion / re-wrapping

			oneOf := []any{
				map[string]string{"type": string(SchemaNull)},
				nonNull,
			}
			return json.Marshal(map[string]any{"oneOf": oneOf})
		}
		// normal schema
		return json.Marshal(v)
	default:
		return nil, fmt.Errorf("invalid SchemaOrRef value type: %T", t.value)
	}
}

func (t *SchemaOrRef) UnmarshalJSON(data []byte) error {
	// Try to unmarshal as reference first
	var refObj struct {
		Ref string `json:"$ref"`
	}
	if err := json.Unmarshal(data, &refObj); err == nil && refObj.Ref != "" {
		t.value = refObj.Ref
		return nil
	}

	// Otherwise, unmarshal as Schema
	var schema Schema
	if err := json.Unmarshal(data, &schema); err != nil {
		return err
	}
	t.value = schema
	return nil
}

func (t SchemaOrRef) IsRef() bool {
	_, ok := t.value.(string)
	return ok
}

func (t SchemaOrRef) GetRef() (string, bool) {
	ref, ok := t.value.(string)
	return ref, ok
}

func (t SchemaOrRef) GetSchema() (Schema, bool) {
	schema, ok := t.value.(Schema)
	return schema, ok
}

type TypedSchema struct {
	Schema SchemaOrRef `json:"schema" yaml:"schema"`
}
