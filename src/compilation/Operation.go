package compilation

type Operation struct {
	Summary     string                  `json:"summary,omitempty" yaml:"summary,omitempty"`
	OperationId string                  `json:"operationId,omitempty" yaml:"operationId,omitempty"`
	Tags        []string                `json:"tags,omitempty" yaml:"tags,omitempty"`
	Parameters  []Parameter             `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	RequestBody *RequestBody            `json:"requestBody,omitempty" yaml:"requestBody,omitempty"`
	Responses   map[StatusCode]Response `json:"responses,omitempty" yaml:"responses,omitempty"`
}
