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

	"github.com/goccy/go-yaml"
	"github.com/masnyjimmy/qapi/compilation"
	"github.com/masnyjimmy/qapi/docs"
	"github.com/masnyjimmy/qapi/swagger"
	"github.com/masnyjimmy/qapi/validation"
	"github.com/rs/cors"
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

var (
	baseUrl *string
	port    *uint16
)

func init() {
	rootCmd.AddCommand(serveCmd)

	baseUrl = serveCmd.Flags().StringP(
		"baseUrl",
		"b",
		"/",
		"Base URL path",
	)

	port = serveCmd.Flags().Uint16P("port", "p", 8080, "Application port")
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

	docBytes, err := os.ReadFile(filename)

	if err != nil {
		return nil, fmt.Errorf("Unable to read file: %w", err)
	}

	if err := validation.Validate(docBytes); err != nil {
		return nil, fmt.Errorf("Validation error: %w", err)
	}

	var document docs.Document

	if err := yaml.Unmarshal(docBytes, &document); err != nil {
		panic(err) // this shouln't happen
	}

	docBytes, err = compilation.CompileToJSON(&document)

	if err != nil {
		return nil, fmt.Errorf("Compilation error: %w", err)
	}

	return docBytes, nil
}

func Serve(input string) {

	document, err := readAPI(input)

	if err != nil {
		log.Fatal(err)
	}

	swaggerHandler, err := swagger.New(document, swagger.Options{
		DebounceTime: swagger.DEFAULT_DEBOUNCE_TIME,
		BaseUrl:      *baseUrl,
	})

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
					log.Printf("Document update failed: %v", err)
					continue
				}

				swaggerHandler.SetDocument(bytes)
				log.Print("Document updated")
			}
		}
		go watchHandler()
	}
	log.Printf("Started server at http://localhost:%v", *port)

	handler := cors.AllowAll().Handler(swaggerHandler.Handler(nil))
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%v", *port), handler))
}
