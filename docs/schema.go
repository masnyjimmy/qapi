package docs

import (
	"fmt"

	"github.com/goccy/go-yaml"
)

// string or object, parsed in compiler

type Property struct {
	Name   string
	Schema Schema
}

type Properties []Property

type Schema struct {
	Value any
}

func (s *Schema) UnmarshalYAML(data []byte) error {

	// First, try to unmarshal as a string
	var str string
	if err := yaml.Unmarshal(data, &str); err == nil {
		s.Value = str
		return nil
	}

	// If not a string, try to unmarshal as an object (map)
	// We need to parse it manually to preserve order
	var rawMap yaml.MapSlice
	if err := yaml.Unmarshal(data, &rawMap); err != nil {
		return fmt.Errorf("failed to unmarshal as string or object: %w", err)
	}

	// Convert MapSlice to Properties to preserve order
	props := make(Properties, 0, len(rawMap))
	for _, item := range rawMap {
		name, ok := item.Key.(string)
		if !ok {
			return fmt.Errorf("property key must be a string, got %T", item.Key)
		}

		// Marshal the value back to YAML and unmarshal into Schema
		valueBytes, err := yaml.Marshal(item.Value)
		if err != nil {
			return fmt.Errorf("failed to marshal property value: %w", err)
		}

		var propSchema Schema
		if err := yaml.Unmarshal(valueBytes, &propSchema); err != nil {
			return fmt.Errorf("failed to unmarshal property schema: %w", err)
		}

		props = append(props, Property{
			Name:   name,
			Schema: propSchema,
		})
	}

	s.Value = props
	return nil
}
