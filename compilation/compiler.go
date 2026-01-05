package compilation

import (
	"encoding/json"
	"fmt"
	"maps"
	"reflect"
	"regexp"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/masnyjimmy/qapi/docs"
)

type CompileContext struct {
	in  *docs.Document
	out *Document

	defaultResponses map[StatusCode]Response
	compiledTraits   map[string]PrecompiledTrait
}

type PrecompiledTrait struct {
	args   []string
	target docs.Trait
}

func (p *PrecompiledTrait) compileSchema(schema docs.Schema, r *strings.Replacer) docs.Schema {
	switch t := schema.(type) {
	case string:
		return r.Replace(t)
	case map[string]docs.Schema:
		for k, v := range t {
			t[k] = p.compileSchema(v, r)
		}
		return t
	default:
		panic(fmt.Errorf("invalid schema underlying type: %v", reflect.TypeOf(schema).Name()))
	}
}

func (p PrecompiledTrait) Compile(values []string) (docs.Trait, error) {
	if len(p.args) != len(values) {
		return docs.Trait{}, fmt.Errorf("Invalid number of values: %v (expected: %v)", len(values), len(p.args))
	}

	var oldnew []string = make([]string, 0)

	for idx, arg := range p.args {
		oldnew = append(oldnew, "#"+arg, values[idx])
	}

	replacer := strings.NewReplacer(oldnew...)

	for idx, params := range p.target.Params {
		p.target.Params[idx].Schema = p.compileSchema(params.Schema, replacer)
	}

	for idx, headers := range p.target.Headers {
		p.target.Headers[idx].Schema = p.compileSchema(headers.Schema, replacer)
	}

	return p.target, nil
}

func MapArray[T ~[]I, U ~[]O, I any, O any](in T, out *U, mapFn func(idx int, in I) O) {
	(*out) = make(U, len(in))

	for idx, val := range in {
		(*out)[idx] = mapFn(idx, val)
	}
}

func newCompileContext(input *docs.Document, output *Document) *CompileContext {
	return &CompileContext{
		in:  input,
		out: output,
	}
}

func (c *CompileContext) CompileInfo() {
	c.out.Info.Title = c.in.Info.Title
	c.out.Info.Version = c.in.Info.Version
}

func (c *CompileContext) CompileServers() {
	MapArray(c.in.Servers, &c.out.Servers, func(idx int, in docs.Server) Server {
		return Server{
			Url:         in.Url,
			Description: in.Description,
		}
	})
}

func (c *CompileContext) CompileTags() {
	MapArray(c.in.Tags, &c.out.Tags, func(idx int, in docs.Tag) Tag {
		return Tag{
			Name:        in.Name,
			Description: in.Description,
		}
	})
}

func (c *CompileContext) ParseSchema(schema docs.Schema) (SchemaOrRef, error) {
	switch v := schema.(type) {
	case string: // expr
		return parseSchema(v)
	case map[string]docs.Schema:
		object := Schema{
			Type:       SchemaObject,
			Properties: make(map[string]SchemaOrRef),
		}
		for name, val := range v {
			if schema, err := c.ParseSchema(val); err != nil {
				return SchemaOrRef{}, err
			} else {
				object.Properties[name] = schema
			}
		}
		return NewSchemaDef(object), nil
	default:
		panic("Invalid schema type")
	}
}

func (c *CompileContext) ParseSchemas() error {

	if c.out.Components.Schemas == nil {
		c.out.Components.Schemas = make(map[string]Schema)
	}

	for name, schema := range c.in.Schemas {
		var schemaOrRef SchemaOrRef
		if sch, err := c.ParseSchema(schema); err != nil {
			return err
		} else {
			schemaOrRef = sch
		}

		schema, ok := schemaOrRef.value.(Schema)
		if !ok {
			return fmt.Errorf("Schema ref when Schema expected")
		}
		c.out.Components.Schemas[name] = schema
	}
	return nil
}

func (c *CompileContext) ParseDefaultResponses() error {

	c.defaultResponses = make(map[StatusCode]Response, len(c.in.DefaultResponses))

	for statusCode, response := range c.in.DefaultResponses {
		outResponse := Response{
			Description: response.Description,
			Content:     make(map[string]TypedSchema),
		}

		for t, sch := range response.TypedSchema {
			schema, err := c.ParseSchema(sch)
			if err != nil {
				return err
			}

			outResponse.Content[t] = TypedSchema{
				Schema: schema,
			}
		}

		c.defaultResponses[statusCode] = outResponse
	}

	return nil
}

var traitEvExpr = regexp.MustCompile(`^([A-Za-z_]\w*)(?:\(\s*([^()]+?)\s*\))?$`)

func (c *CompileContext) compileTrait(args string, t docs.Trait) (PrecompiledTrait, error) {
	splited := strings.Split(args, ",")
	argsOut := make([]string, len(splited))

	for idx, arg := range splited {
		arg = strings.TrimSpace(arg)
		argsOut[idx] = arg
	}

	return PrecompiledTrait{
		args:   argsOut,
		target: t,
	}, nil
}

func (c *CompileContext) compileTraits() error {
	c.compiledTraits = make(map[string]PrecompiledTrait, len(c.in.Traits))

	for expr, trait := range c.in.Traits {
		exprGrp := traitEvExpr.FindStringSubmatch(expr)
		if exprGrp == nil {
			return fmt.Errorf("invalid trait definition expression: %v", expr)
		}
		ident := exprGrp[1]
		args := exprGrp[2]

		result, err := c.compileTrait(args, trait)

		if err != nil {
			return err
		}

		c.compiledTraits[ident] = result
	}
	return nil
}

func (c *CompileContext) evaluateTrait(expr string) (docs.Trait, error) {
	groups := traitEvExpr.FindStringSubmatch(expr)

	if groups == nil {
		return docs.Trait{}, fmt.Errorf("invalid trait evaluate expression: %v\n expected: ident[(arg(,args)...)]", expr)
	}

	ident := groups[1]
	params := groups[2]

	trait, has := c.compiledTraits[ident]

	if !has {
		return docs.Trait{}, fmt.Errorf("no %v trait found", ident)
	}

	values := make([]string, 0)

	for param := range strings.SplitSeq(params, ",") {
		param = strings.TrimSpace(param)
		values = append(values, param)
	}

	return trait.Compile(values)

}

func (c *CompileContext) evaluateTraits(traits []string) ([]docs.Trait, error) {
	if traits == nil {
		return nil, nil
	}

	var out []docs.Trait = make([]docs.Trait, len(traits))

	for idx, in := range traits {
		res, err := c.evaluateTrait(in)
		if err != nil {
			return nil, err
		}
		out[idx] = res
	}

	return out, nil
}

func (c *CompileContext) parseMethod(method *docs.Method, tags []string, path string) (*Operation, error) {
	if method == nil {
		return nil, nil
	}

	makeParam := func(p *docs.Param, in ParamIn) (Parameter, error) {
		// query is path in {name} in path
		if in == InQuery && strings.Contains(path, "{"+p.Name+"}") {
			in = InPath
		}

		schema, err := c.ParseSchema(p.Schema)
		if err != nil {
			return Parameter{}, err
		}

		return Parameter{
			Name:     p.Name,
			In:       in,
			Required: p.Required,
			Schema:   schema,
		}, nil
	}

	traits, err := c.evaluateTraits(method.Traits)

	if err != nil {
		return nil, err
	}

	out := Operation{
		Summary: method.Description,
		// TODO: add OperationId: "asd" ,
		Tags:       tags,
		Parameters: make([]Parameter, 0),
		Responses:  maps.Clone(c.defaultResponses),
	}

	// params to parameters

	if method.Params != nil {
		for _, v := range method.Params {
			outParam, err := makeParam(&v, InQuery)

			if err != nil {
				return nil, err
			}

			out.Parameters = append(out.Parameters, outParam)
		}
	}

	// headers to parameters

	if method.Headers != nil {
		for _, header := range method.Headers {

			outParam, err := makeParam(&header, InHeader)
			if err != nil {
				return nil, err
			}

			out.Parameters = append(out.Parameters, outParam)
		}
	}

	// put trait's params / headers into operation

	for _, t := range traits {
		for _, param := range t.Params {
			outParam, err := makeParam(&param, InQuery)
			if err != nil {
				return nil, err
			}
			out.Parameters = append(out.Parameters, outParam)
		}
		for _, header := range t.Headers {
			outHeader, err := makeParam(&header, InHeader)
			if err != nil {
				return nil, err
			}
			out.Parameters = append(out.Parameters, outHeader)
		}
	}

	if method.Body != nil {
		body := RequestBody{
			Required: true,
			Content:  make(map[string]TypedSchema, len(method.Body)),
		}

		for t, s := range method.Body {
			schema, err := c.ParseSchema(s)
			if err != nil {
				return nil, err
			}

			body.Content[t] = TypedSchema{
				Schema: schema,
			}
		}

		out.RequestBody = &body
	}

	for statusCode, response := range method.Responses {
		outResponse := Response{
			Description: response.Description,
			Content:     make(map[string]TypedSchema),
		}

		for mediaType, schema := range response.TypedSchema {
			outSchema, err := c.ParseSchema(schema)
			if err != nil {
				return nil, err
			}
			outResponse.Content[mediaType] = TypedSchema{
				Schema: outSchema,
			}
		}

		out.Responses[statusCode] = outResponse
	}

	return &out, nil
}

func (c *CompileContext) ParseEndpoints() error {
	c.out.Paths = make(map[string]Path, len(c.in.Endpoints))

	var nameToGroup map[string]string = make(map[string]string)

	// collect groups names
	for path, endpoint := range c.in.Endpoints {
		group, isGroup := endpoint.AsGroup()

		if !isGroup {
			continue
		}

		nameToGroup[group.AsGroup] = path
	}

	// build openapi paths from endpoints

	for path, endpoint := range c.in.Endpoints {
		endpoint, isEndpoint := endpoint.AsFinal()

		if !isEndpoint {
			continue
		}

		if endpoint.InGroup != "" {

			groupPath, has := nameToGroup[endpoint.InGroup]

			if !has {
				return fmt.Errorf("Invalid inGroup value, %s is not group", endpoint.InGroup)
			}

			path = groupPath + path
			group, _ := c.in.Endpoints[groupPath].AsGroup()
			endpoint.Tags = append(endpoint.Tags, group.Tags...)

			for group.InGroup != "" {
				groupPath, has = nameToGroup[group.InGroup]

				if !has {
					return fmt.Errorf("Invalid %s group inGroup value, %s is not group", group.AsGroup, group.InGroup)
				}

				path = groupPath + path
				group, _ = c.in.Endpoints[groupPath].AsGroup()
				endpoint.Tags = append(endpoint.Tags, group.Tags...)
			}
		}

		outPath := Path{}

		if get, err := c.parseMethod(endpoint.Get, endpoint.Tags, path); err != nil {
			return err
		} else {
			outPath.Get = get
		}

		setMethod := func(outOp **Operation, met *docs.Method) error {
			op, err := c.parseMethod(met, endpoint.Tags, path)
			if err != nil {
				return err
			}

			*outOp = op
			return nil
		}

		if err := setMethod(&outPath.Get, endpoint.Get); err != nil {
			return err
		}

		if err := setMethod(&outPath.Put, endpoint.Put); err != nil {
			return err
		}

		if err := setMethod(&outPath.Post, endpoint.Post); err != nil {
			return err
		}

		if err := setMethod(&outPath.Delete, endpoint.Delete); err != nil {
			return err
		}

		c.out.Paths[path] = outPath
	}

	return nil
}

func (c *CompileContext) Parse() error {
	c.CompileInfo()

	c.CompileServers()

	c.CompileTags()

	if err := c.ParseSchemas(); err != nil {
		return err
	}

	if err := c.ParseDefaultResponses(); err != nil {
		return err
	}

	if err := c.compileTraits(); err != nil {
		return err
	}

	if err := c.ParseEndpoints(); err != nil {
		return err
	}

	return nil
}

func Compile(out *Document, in *docs.Document) error {
	ctx := newCompileContext(in, out)

	if err := ctx.Parse(); err != nil {
		return fmt.Errorf("Compilation error: %w", err)
	}

	return nil
}

func CompileToJSON(in *docs.Document) ([]byte, error) {
	out := Document{
		Openapi: "3.1.0",
	}

	if err := Compile(&out, in); err != nil {
		return nil, err
	}

	bytes, err := json.Marshal(out)

	if err != nil {
		return nil, fmt.Errorf("Unable to produce json document: %w", err)
	}

	return bytes, nil
}

func CompileToYAML(in *docs.Document) ([]byte, error) {
	out := Document{
		Openapi: "3.1.0",
	}

	if err := Compile(&out, in); err != nil {
		return nil, err
	}

	bytes, err := yaml.Marshal(out)

	if err != nil {
		return nil, fmt.Errorf("Unable to produce yaml document: %w", err)
	}

	return bytes, nil
}
