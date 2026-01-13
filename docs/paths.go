package docs

import "github.com/goccy/go-yaml"

type Path struct {
	Tags   []string        `yaml:"tags,omitempty"`
	Get    *Method         `yaml:"get,omitempty"`
	Post   *Method         `yaml:"post,omitempty"`
	Put    *Method         `yaml:"put,omitempty"`
	Patch  *Method         `yaml:"patch,omitempty"`
	Delete *Method         `yaml:"delete,omitempty"`
	Nested map[string]Path `yaml:",inline"`
}

type Paths = map[string]Path

func (p *Path) UnmarshalYAML(bytes []byte) error {

	var raw map[string]yaml.RawMessage

	if err := yaml.Unmarshal(bytes, &raw); err != nil {
		return err
	}

	if tags, has := raw["tags"]; has {
		if err := yaml.Unmarshal(tags, &p.Tags); err != nil {
			return err
		}
		delete(raw, "tags")
	}

	if get, has := raw["get"]; has {
		p.Get = new(Method)
		if err := yaml.Unmarshal(get, p.Get); err != nil {
			return err
		}
		delete(raw, "get")
	}

	if put, has := raw["put"]; has {
		p.Put = new(Method)
		if err := yaml.Unmarshal(put, p.Put); err != nil {
			return err
		}
		delete(raw, "put")
	}

	if post, has := raw["post"]; has {
		p.Post = new(Method)
		if err := yaml.Unmarshal(post, p.Post); err != nil {
			return err
		}
		delete(raw, "post")
	}

	if patch, has := raw["patch"]; has {
		p.Patch = new(Method)
		if err := yaml.Unmarshal(patch, p.Patch); err != nil {
			return err
		}
		delete(raw, "patch")
	}

	if del, has := raw["delete"]; has {
		p.Delete = new(Method)
		if err := yaml.Unmarshal(del, p.Delete); err != nil {
			return err
		}
		delete(raw, "delete")
	}

	if len(raw) == 0 {
		return nil
	} else {
		p.Nested = make(map[string]Path)
	}

	for k, v := range raw {
		var outPath Path
		if err := yaml.Unmarshal(v, &outPath); err != nil {
			return err
		}
		p.Nested[k] = outPath
	}

	return nil

}
