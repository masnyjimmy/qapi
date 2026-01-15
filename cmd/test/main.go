package main

import (
	"fmt"
	"os"

	"github.com/goccy/go-yaml"
	"github.com/masnyjimmy/qapi/compilation"
	"github.com/masnyjimmy/qapi/docs"
)

func main() {
	bytes, err := os.ReadFile("private/api.yaml")

	if err != nil {
		panic(err)
	}
	var document docs.Document

	if err := yaml.Unmarshal(bytes, &document); err != nil {
		panic(err)
	}

	var out compilation.Document

	if err := compilation.Compile(&out, &document); err != nil {
		panic(err)
	}

	fmt.Println(out)
}
