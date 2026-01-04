package docs

type Param struct {
	Name     string `yaml:"name"`
	Schema   Schema `yaml:"schema"`
	Required bool   `yaml:"required"`
}

type Params = []Param
