package docs

import (
	"github.com/goccy/go-yaml"
)

type Response struct {
	Description string `yaml:"description"`
	TypedSchema `yaml:"-"`
}

type StatusCode = string

type Responses = map[StatusCode]Response

// UnmarshalYAML implements BytesUnmarshaler for goccy/go-yaml
func (r *Response) UnmarshalYAML(data []byte) error {
	// First unmarshal into a map to get all fields
	var raw map[string]yaml.RawMessage
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return err
	}

	// Extract description
	if desc, ok := raw["description"]; ok {
		if err := yaml.Unmarshal(desc, &r.Description); err != nil {
			return err
		}

		delete(raw, "description")
	}

	// Initialize the map if needed
	if r.TypedSchema == nil {
		r.TypedSchema = make(TypedSchema)
	}

	// All remaining fields are media types with schemas
	for schemaType, schemaData := range raw {
		var out Schema
		if err := yaml.Unmarshal(schemaData, &out); err != nil {
			return err
		}
		r.TypedSchema[schemaType] = out
	}

	return nil
}
