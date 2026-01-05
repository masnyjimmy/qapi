package docs

type Document struct {
	Info             Info              `yaml:"info"`
	Servers          []Server          `yaml:"servers"`
	Tags             []Tag             `yaml:"tags,omitempty"`
	Schemas          map[string]Schema `yaml:"schemas,omitempty"`
	Traits           Traits            `yaml:"traits,omitempty"`
	DefaultResponses Responses         `yaml:"defaultResponses,omitempty"`
	Paths            Paths             `yaml:"paths,omitempty"`
}
