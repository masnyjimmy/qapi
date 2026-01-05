package docs

type Path struct {
	Tags   []string        `yaml:"tags,omitempty"`
	Get    *Method         `yaml:"get,omitempty"`
	Post   *Method         `yaml:"post,omitempty"`
	Put    *Method         `yaml:"put,omitempty"`
	Delete *Method         `yaml:"delete,omitempty"`
	Nested map[string]Path `yaml:",inline"`
}

type Paths = map[string]Path
