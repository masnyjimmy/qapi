package validation

import (
	_ "embed"
	"encoding/json"
	"fmt"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

//go:embed schema.json
var schemaBytes []byte

var schema *jsonschema.Schema

func init() {
	compiler := jsonschema.NewCompiler()
	compiler.DefaultDraft(jsonschema.Draft2020)

	var object any

	if err := json.Unmarshal(schemaBytes, &object); err != nil {
		panic(err)
	}

	if err := compiler.AddResource("qapi-schema.json", object); err != nil {
		panic(err)
	}

	schema = compiler.MustCompile("qapi-schema.json")
}

func Validate(documentBytes []byte) error {
	var document any

	if err := json.Unmarshal(documentBytes, &document); err != nil {
		return fmt.Errorf("Unable to parse document: %w", err)
	}

	return schema.Validate(document)
}
