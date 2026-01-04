package validation

import "github.com/santhosh-tekuri/jsonschema/v6"

type SchemaValidator struct {
	schema *jsonschema.Schema
}

func New(schemaUrl string) *SchemaValidator {
	compiler := jsonschema.NewCompiler()
	compiler.DefaultDraft(jsonschema.Draft2020)

	return &SchemaValidator{
		schema: compiler.MustCompile(schemaUrl),
	}
}

func (self *SchemaValidator) ValidateObject(obj any) error {
	return self.schema.Validate(obj)
}
