package docs

type Tag struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description,omitempty"`
}
