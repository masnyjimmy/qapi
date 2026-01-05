package docs

import (
	"github.com/goccy/go-yaml"
)

type EndpointGroup struct {
	AsGroup string   `yaml:"asGroup"`
	InGroup string   `yaml:"inGroup,omitempty"`
	Tags    []string `yaml:"tags,omitempty"`
}

type EndpointFinal struct {
	InGroup string   `yaml:"inGroup,omitempty"`
	Tags    []string `yaml:"tags,omitempty"`
	Get     *Method  `yaml:"get,omitempty"`
	Post    *Method  `yaml:"post,omitempty"`
	Put     *Method  `yaml:"put,omitempty"`
	Delete  *Method  `yaml:"delete,omitempty"`
}

type Endpoint struct {
	value any
}

func (e Endpoint) AsGroup() (out EndpointGroup, ok bool) {
	out, ok = e.value.(EndpointGroup)
	return
}

func (e Endpoint) AsFinal() (out EndpointFinal, ok bool) {
	out, ok = e.value.(EndpointFinal)
	return
}

func checkIsGroupEndpoint(data []byte) bool {
	var asGroupHolder struct {
		AsGroup string `yaml:"asGroup"`
	}

	if err := yaml.Unmarshal(data, &asGroupHolder); err != nil {
		panic(err)
	}

	if asGroupHolder.AsGroup == "" {
		return false
	}

	return true
}

func (e *Endpoint) UnmarshalYAML(data []byte) error {

	isGroup := checkIsGroupEndpoint(data)

	if isGroup {
		var out EndpointGroup

		if err := yaml.Unmarshal(data, &out); err != nil {
			return err
		}

		e.value = out
	} else {
		var out EndpointFinal

		if err := yaml.Unmarshal(data, &out); err != nil {
			return err
		}

		e.value = out
	}

	return nil
}

type Endpoints = map[string]Endpoint
