package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/goccy/go-yaml"
	"github.com/masnyjimmy/qapi/compilation"
	"github.com/masnyjimmy/qapi/docs"
)

func main() {
	bytes, _ := os.ReadFile("private/api.yaml")

	var document docs.Document

	if err := yaml.Unmarshal(bytes, &document); err != nil {
		panic(err)
	}

	var outDocument compilation.Document

	compilation.Compile(&outDocument, &document)

	bytes, _ = json.MarshalIndent(document.Paths, "", "    ")

	fmt.Print(string(bytes))
}
