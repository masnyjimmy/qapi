package compilation

type Response struct {
	Description string                 `json:"description,omitempty" yaml:"description,omitempty"`
	Content     map[string]TypedSchema `json:"content" yaml:"content"`
}

type StatusCode = string
