package docs

type Method struct {
	Description string      `yaml:"description,omitempty"`
	Traits      []string    `yaml:"traits,omitempty"`
	Params      Params      `yaml:"params,omitempty"`
	Headers     Params      `yaml:"headers,omitempty"`
	Body        TypedSchema `yaml:"body,omitempty"`
	Responses   Responses   `yaml:"responses,omitempty"`
}
