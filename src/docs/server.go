package docs

type Server struct {
	Url         string `yaml:"url"`
	Description string `yaml:"description,omitempty"`
}
