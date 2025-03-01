package main

import (
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"text/template"
)

// dotembed is a little tool to tackle the shortcomings of go:embed described in golang/go#43854.
// tldr: go:embed does not include dotfiles which is critical for the template that needs to
// be included in the go/template binary.
// dotembed goes trough all files in a given path and creates one big go:embed directive that explicitly
// includes all otherwise excluded dotfiles.
func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "an error occurred: %s\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	var (
		targetDir  string
		outputFile string
	)

	config := struct {
		EmbedPaths   map[string]struct{}
		Package      string
		VariableName string
	}{
		EmbedPaths: map[string]struct{}{},
	}

	flag.StringVar(&targetDir, "target", "", "The dir to generate the embed.FS for including all files")
	flag.StringVar(&outputFile, "o", "./embed_fulldir.go", "The file to write")
	flag.StringVar(&config.Package, "pkg", "main", "The resulting file's package")
	flag.StringVar(&config.VariableName, "var", "Files", "The variable name for the embed.FS")

	flag.CommandLine.Parse(args)

	if targetDir == "" {
		return errors.New("target is a required parameter")
	}

	config.EmbedPaths[targetDir] = struct{}{}
	filepath.WalkDir(targetDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		file := filepath.Base(path)
		if len([]rune(file)) > 1 && (strings.HasPrefix(file, ".")) {
			config.EmbedPaths[path] = struct{}{}
		}

		return nil
	})

	file, err := os.Create(outputFile)
	if err != nil {
		return err
	}

	return tmpl.Execute(file, config)
}

func keys(set map[string]struct{}) []string {
	strSlice := make([]string, 0, len(set))

	for k := range set {
		strSlice = append(strSlice, k)
	}

	return strSlice
}

func sortStrings(slice []string) []string {
	sorted := slice
	sort.Strings(sorted)
	return sorted
}

var (
	tmplString = `package {{ .Package }}

import "embed"

//go:embed {{ join (sort (.EmbedPaths | keys)) " " }}
var {{ .VariableName }} embed.FS
`
	tmpl = template.Must(
		template.New("").Funcs(template.FuncMap{
			"join": strings.Join,
			"keys": keys,
			"sort": sortStrings,
		}).Parse(tmplString),
	)
)
