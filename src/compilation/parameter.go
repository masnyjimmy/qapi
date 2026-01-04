package compilation

type ParamIn string

const (
	InPath   ParamIn = "path"
	InQuery  ParamIn = "query"
	InHeader ParamIn = "header"
)

type Parameter struct {
	Name     string      `json:"name" yaml:"name"`
	In       ParamIn     `json:"in" yaml:"in"`
	Required bool        `json:"required" yaml:"required"`
	Schema   SchemaOrRef `json:"schema" yaml:"schema"`
}
