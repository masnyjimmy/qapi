package docs

type Trait struct {
	Params  Params `yaml:"params,omitempty"`
	Headers Params `yaml:"headers,omitempty"`
}

type Traits = map[string]Trait
