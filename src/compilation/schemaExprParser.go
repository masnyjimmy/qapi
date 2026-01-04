package compilation

import (
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
)

var schemaExprRegex = regexp.MustCompile(`^(?:((?:boolean|string|integer|number)\??)(?:\((.*)\))?|(?:<(\w+)>))(\??\[[^\]]*\])*$`)

func extractBetween(s string, left, right string) (string, bool) {
	s, ok := strings.CutPrefix(s, left)
	if !ok {
		return "", false
	}
	return strings.CutSuffix(s, right)
}

func parseArraySize(expr string) (min, max uint, err error) {
	opPos := strings.Index(expr, ":")
	if opPos == -1 {
		val, err := strconv.ParseUint(expr, 10, 64)
		if err != nil {
			return 0, 0, err
		}
		return 0, uint(val), nil
	}

	minVal, err := strconv.ParseUint(expr[:opPos], 10, 64)
	if err != nil {
		return 0, 0, err
	}

	maxVal, err := strconv.ParseUint(expr[opPos+1:], 10, 64)
	if err != nil {
		return 0, 0, err
	}

	return uint(minVal), uint(maxVal), nil
}

func applyArrayParams(schema *Schema, arrExpr string) error {
	if arrExpr == "" {
		return nil
	}

	params := strings.Split(arrExpr, ",")

	for _, param := range params {
		param = strings.TrimSpace(param)
		if param == "" {
			continue
		}

		switch param {
		case "*":
			schema.UniqueItems = true
		default:
			min, max, err := parseArraySize(param)
			if err != nil {
				return err
			}
			schema.MinItems = &min
			schema.MaxItems = &max
		}
	}

	return nil
}

func parseArraySchema(element SchemaOrRef, reader *strings.Reader) (Schema, error) {
	ch, _, err := reader.ReadRune()

	if err != nil {
		return Schema{}, err
	}

	schema := Schema{
		Type:  SchemaArray,
		Items: &element,
	}

	// first can be optional ? which means its nullable
	if ch == '?' {
		schema.nullable = true
		ch, _, err = reader.ReadRune()
		if err != nil {
			return Schema{}, err
		}
	}
	// then read [] content
	if ch != '[' {
		return Schema{}, fmt.Errorf("invalid glyph found: %v", ch)
	}

	var result strings.Builder
	for {
		ch, _, err := reader.ReadRune()
		if err == io.EOF {
			return Schema{}, fmt.Errorf("] not found, reached EOF")
		}
		if err != nil {
			return Schema{}, err
		}

		if ch == ']' {
			break
		}

		if _, err := result.WriteRune(ch); err != nil {
			return Schema{}, err
		}
	}

	if err := applyArrayParams(&schema, result.String()); err != nil {
		return Schema{}, err
	}

	if reader.Len() != 0 {
		return parseArraySchema(NewSchemaDef(schema), reader)
	}

	return schema, nil
}

func valToPtr[T any](value T) *T {
	return &value
}

func parseRange(schema *Schema, param string) error {
	idx := strings.IndexAny(param, "<>:")

	if idx == -1 {
		return fmt.Errorf("range operator not found")
	}

	var left, right *int

	// Parse left side
	if idx > 0 {
		val, err := strconv.ParseInt(param[:idx], 10, 64)
		if err != nil {
			return err
		}
		left = valToPtr(int(val))
	}

	// Parse right side
	if idx < len(param)-1 {
		val, err := strconv.ParseInt(param[idx+1:], 10, 64)
		if err != nil {
			return err
		}
		right = valToPtr(int(val))
	}

	// Apply based on operator
	switch param[idx] {
	case '<', ':':
		schema.Minimum = left
		schema.Maximum = right
	case '>':
		schema.Minimum = right
		schema.Maximum = left
	default:
		return fmt.Errorf("unknown range operator: %c", param[idx])
	}

	return nil
}

func splitAt(s string, idx int) (string, string) {
	return s[:idx], s[idx:]
}

func parseObjectSchema(t string, params string) (Schema, error) {
	t, nullable := strings.CutSuffix(t, "?")

	out := Schema{
		Type:     SchemaType(t),
		nullable: nullable,
	}

	handleSigned := func(p string) bool {
		if len(p) < 2 {
			return false
		}
		sign, rest := splitAt(p, 1)
		switch sign {
		case "$":
			out.Format = rest
			return true
		}
		return false
	}

	handleDefault := func(p string) (bool, error) {
		// Parse based on schema type
		if p == "null" {
			out.Default = valToPtr(any(nil))
			return true, nil
		}
		switch out.Type {
		case "integer":
			val, err := strconv.ParseInt(p, 10, 64)
			if err != nil {
				return false, fmt.Errorf("invalid integer default value: %s", p)
			}
			out.Default = valToPtr(any(int(val)))
		case "number":
			val, err := strconv.ParseFloat(p, 64)
			if err != nil {
				return false, fmt.Errorf("invalid number default value: %s", p)
			}
			out.Default = valToPtr(any(val))
		case "boolean":
			switch p {
			case "true":
				out.Default = valToPtr(any(true))
			case "false":
				out.Default = valToPtr(any(false))
			default:
				return false, fmt.Errorf("invalid boolean default value: %s", p)
			}
		case "string":
			if s, ok := extractBetween(p, "\"", "\""); ok {
				out.Default = valToPtr(any(s))
			} else {
				return false, fmt.Errorf("invalid string default value: %s", p)
			}
		default:
			return false, fmt.Errorf("cannot set default for type: %s", out.Type)
		}
		return true, nil
	}

	if params == "" {
		return out, nil
	}

	for _, param := range strings.Split(params, ",") {
		param = strings.TrimSpace(param)
		if param == "" {
			continue
		}

		if handleSigned(param) {
			continue
		}

		if strings.ContainsAny(param, "<>:") {
			if err := parseRange(&out, param); err != nil {
				return Schema{}, err
			}
			continue
		}

		// Try to handle as default value
		handled, err := handleDefault(param)
		if err != nil {
			return Schema{}, err
		}
		if handled {
			continue
		}

		return Schema{}, fmt.Errorf("unknown parameter: %s", param)
	}

	return out, nil
}

func parseSchema(expr string) (SchemaOrRef, error) {
	sub := schemaExprRegex.FindStringSubmatch(expr)
	if sub == nil {
		return SchemaOrRef{}, fmt.Errorf("invalid schema expression: %s", expr)
	}

	baseType := sub[1] // boolean, string, integer, number
	params := sub[2]   // parameters in parentheses
	ref := sub[3]      // reference like <Type>
	arr := sub[4]      // array notation

	var out SchemaOrRef

	if ref != "" {
		out = NewSchemaRef(fmt.Sprintf("#/components/schemas/%s", ref))
	} else {
		schema, err := parseObjectSchema(baseType, params)
		if err != nil {
			return SchemaOrRef{}, err
		}
		out = NewSchemaDef(schema)
	}

	if arr != "" {
		arrReader := strings.NewReader(arr)
		schema, err := parseArraySchema(out, arrReader)
		if err != nil {
			return SchemaOrRef{}, err
		}
		return NewSchemaDef(schema), nil
	}

	return out, nil
}

func ParseSchemaWithContext(expr string, context map[string]string) (SchemaOrRef, error) {
	oldnew := make([]string, 0, len(context)*2)
	for old, new := range context {
		oldnew = append(oldnew, "#"+old, new)
	}

	replacer := strings.NewReplacer(oldnew...)
	expr = replacer.Replace(expr)

	return parseSchema(expr)
}
