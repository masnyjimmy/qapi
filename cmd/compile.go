/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"log"
	"os"
	"path/filepath"

	"github.com/goccy/go-yaml"
	"github.com/masnyjimmy/qapi/src/compilation"
	"github.com/masnyjimmy/qapi/src/docs"
	"github.com/masnyjimmy/qapi/src/validation"
	"github.com/spf13/cobra"
)

// compileCmd represents the compile command
var compileCmd = &cobra.Command{
	Use:   "compile",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {

		output, _ := cmd.Flags().GetString("output")
		input, _ := cmd.Flags().GetString("input")

		if res := CompileFile(output, input); res != 0 {
			os.Exit(res)
		}
	},
}

var errorLogger *log.Logger = log.New(os.Stderr, "Error", log.Ltime)

func validate(bytes []byte) error {
	var object any

	if err := yaml.Unmarshal(bytes, &object); err != nil {
		return err
	}

	validator := validation.New("schema.json")

	return validator.ValidateObject(object)
}

func CompileFile(output, input string) int {

	log.Printf("Reading %v", input)

	bytes, err := os.ReadFile(input)

	if err != nil {
		errorLogger.Printf("Unable to read file \"%v\": %v", input, err)
		return 1
	}

	log.Print("Validating schema..")

	if err := validate(bytes); err != nil {
		errorLogger.Printf("Validation failed: %v", err)
		return 2
	}

	log.Printf("Parsing document..")

	var document docs.Document

	if err := yaml.Unmarshal(bytes, &document); err != nil {
		errorLogger.Printf("Unable to parse document: %v", err)
		return 3
	}

	log.Print("Compiling api document..")

	ext := filepath.Ext(output)

	log.Printf("Output file extension: %v", ext)

	switch ext {
	case ".json":
		log.Printf("Type selected: json")
		bytes, err = compilation.CompileToJSON(&document)
	case ".yaml":
		log.Printf("Type selected: yaml")
		bytes, err = compilation.CompileToYAML(&document)
	default:
		log.Printf("Unkown file extension, selecting yaml")
		bytes, err = compilation.CompileToYAML(&document)
	}

	log.Printf("Writing to %v", output)

	if err := os.WriteFile(output, bytes, 0644); err != nil {
		errorLogger.Printf("Unable to write file %v: %v", output, err)
		return 4
	}

	log.Printf("Finished succesfully :)")
	return 0
}

func init() {
	rootCmd.AddCommand(compileCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// compileCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// compileCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

	compileCmd.Flags().StringP("output", "o", "openapi.yaml", "Output filepath")
	compileCmd.MarkFlagRequired("output")
	compileCmd.MarkFlagFilename("output", "yaml", "json")

}
