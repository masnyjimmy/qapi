package compilation

type RequestBody struct {
	Required    bool                   `json:"required,omitempty" yaml:"required,omitempty"`
	Description string                 `json:"description,omitempty" yaml:"description,omitempty"`
	Content     map[string]TypedSchema `json:"content" yaml:"content"`
}
