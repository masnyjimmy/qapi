package compilation

type Document struct {
	Openapi    string          `json:"openapi" yaml:"openapi"`
	Info       Info            `json:"info" yaml:"info"`
	Servers    []Server        `json:"servers,omitempty"`
	Tags       Tags            `json:"tags" yaml:"tags"`
	Components Components      `json:"components,omitempty" yaml:"components,omitempty"`
	Paths      map[string]Path `json:"paths,omitempty" yaml:"paths,omitempty"`
}
