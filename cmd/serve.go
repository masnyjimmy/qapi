/*
Copyright © 2026 NAME HERE
*/
package cmd

import (
	_ "embed"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/masnyjimmy/qapi/src/compilation"
	"github.com/masnyjimmy/qapi/src/docs"
	"github.com/masnyjimmy/qapi/src/swagger"
	"github.com/spf13/cobra"
)

// ==================== Cobra Command ====================

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Serve OpenAPI documentation with Redoc",
	Run: func(cmd *cobra.Command, _ []string) {
		input, err := cmd.Flags().GetString("input")
		if err != nil {
			panic(err)
		}
		Serve(input)
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)

	serveCmd.Flags().StringP(
		"input",
		"i",
		"openapi.yaml",
		"OpenAPI 3.1 YAML file to watch",
	)
}

/*
When update
1. read bytes
2. validate schema
3. unmarshal
4. compile
5. marshal
*/
func readAPI(filename string) ([]byte, error) {

	bytes, err := os.ReadFile(filename)

	if err != nil {
		return nil, fmt.Errorf("Unable to read file: %w", err)
	}

	if err := validate(bytes); err != nil {
		return nil, fmt.Errorf("Validation error: %w", err)
	}

	var document docs.Document

	bytes, err = compilation.CompileToJSON(&document)

	if err != nil {
		return nil, fmt.Errorf("Compilation error: %w", err)
	}

	return bytes, nil
}

func Serve(input string) {

	document, err := readAPI(input)

	if err != nil {
		log.Fatal(err)
	}

	swaggerHandler, err := swagger.New(document, swagger.DefaultOptions())

	if err != nil {
		log.Fatalf("Invalid input: %v", err)
	}

	watcher, err := swagger.WatchFile(input, swagger.DEFAULT_DEBOUNCE_TIME)

	if err != nil {
		log.Printf("Unable to watch for file updates: %v", err)
	} else {
		watchHandler := func() {
			for err := range watcher.Update {
				if err != nil {
					log.Print(err)
					continue
				}

				bytes, err := readAPI(input)
				if err != nil {
					log.Printf("Unable to update api: %v", err)
					continue
				}

				swaggerHandler.SetDocument(bytes)
			}
		}
		go watchHandler()
	}
	log.Print("Started server at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", swaggerHandler.Handler(nil)))
}
