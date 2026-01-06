// Package main runs the commentlint CLI.
package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

type pkgInfo struct {
	Dir         string   `json:"Dir"`
	GoFiles     []string `json:"GoFiles"`
	TestGoFiles []string `json:"TestGoFiles"`
}

type finding struct {
	pos token.Position
	msg string
}

type golangciConfig struct {
	Issues struct {
		MaxIssuesPerLinter int      `yaml:"max-issues-per-linter"`
		ExcludeDirs        []string `yaml:"exclude-dirs"`
		ExcludeFiles       []string `yaml:"exclude-files"`
	} `yaml:"issues"`
}

// main is the entrypoint for the comment linter CLI.
func main() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [packages]\n", os.Args[0])
		fmt.Fprintf(flag.CommandLine.Output(), "Ensures every function has a doc comment. Defaults to ./...\n")
		flag.PrintDefaults()
	}
	flag.Parse()
	patterns := flag.Args()
	if len(patterns) == 0 {
		patterns = []string{"./..."}
	}

	cfg, err := loadConfig(".golangci.yml")
	if err != nil {
		fmt.Fprintf(os.Stderr, "commentlint: %v\n", err)
		os.Exit(1)
	}

	pkgs, err := listPackages(patterns)
	if err != nil {
		fmt.Fprintf(os.Stderr, "commentlint: %v\n", err)
		os.Exit(1)
	}

	excludeDirs := normaliseDirs(cfg.Issues.ExcludeDirs)
	excludeRegex, err := compileRegexps(cfg.Issues.ExcludeFiles)
	if err != nil {
		fmt.Fprintf(os.Stderr, "commentlint: %v\n", err)
		os.Exit(1)
	}

	limit := cfg.Issues.MaxIssuesPerLinter

	fset := token.NewFileSet()
	var findings []finding
	truncated := false
	for _, pkg := range pkgs {
		files := append([]string{}, pkg.GoFiles...)
		files = append(files, pkg.TestGoFiles...)
		for _, file := range files {
			filename := filepath.Join(pkg.Dir, file)
			rel := filepath.ToSlash(relativePath(filename))
			if shouldExclude(rel, excludeDirs, excludeRegex) {
				continue
			}
			if isGeneratedFile(filename) {
				continue
			}
			f, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
			if err != nil {
				fmt.Fprintf(os.Stderr, "commentlint: parse %s: %v\n", filename, err)
				os.Exit(1)
			}
			for _, decl := range f.Decls {
				fn, ok := decl.(*ast.FuncDecl)
				if !ok {
					continue
				}
				if fn.Body == nil {
					continue
				}
				if fn.Doc == nil || strings.TrimSpace(fn.Doc.Text()) == "" {
					pos := fset.Position(fn.Pos())
					findings = append(findings, finding{pos: pos, msg: fmt.Sprintf("missing doc comment for function %q", fn.Name.Name)})
					if limit > 0 && len(findings) >= limit {
						truncated = true
						break
					}
				}
			}
			if truncated {
				break
			}
		}
		if truncated {
			break
		}
	}

	if len(findings) > 0 {
		for _, f := range findings {
			fmt.Fprintf(os.Stderr, "%s:%d:%d: %s\n", relativePath(f.pos.Filename), f.pos.Line, f.pos.Column, f.msg)
		}
		if truncated && limit > 0 {
			fmt.Fprintf(os.Stderr, "commentlint: output truncated after %d issues (see .golangci.yml)\n", limit)
		}
		os.Exit(1)
	}
}

// loadConfig carga la configuración de golangci-lint desde el archivo YAML especificado.
func loadConfig(path string) (golangciConfig, error) {
	var cfg golangciConfig
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, nil
		}
		return cfg, fmt.Errorf("read %s: %w", path, err)
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parse %s: %w", path, err)
	}
	return cfg, nil
}

// normaliseDirs normaliza una lista de directorios, eliminando prefijos y convirtiéndolos a barras diagonales.
func normaliseDirs(dirs []string) []string {
	out := make([]string, 0, len(dirs))
	for _, d := range dirs {
		d = strings.TrimSpace(strings.TrimPrefix(d, "./"))
		if d == "" {
			continue
		}
		out = append(out, filepath.ToSlash(d))
	}
	return out
}

// compileRegexps compila una lista de patrones de expresiones regulares.
func compileRegexps(patterns []string) ([]*regexp.Regexp, error) {
	var out []*regexp.Regexp
	for _, p := range patterns {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		rx, err := regexp.Compile(p)
		if err != nil {
			return nil, fmt.Errorf("invalid exclude regex %q: %w", p, err)
		}
		out = append(out, rx)
	}
	return out, nil
}

// listPackages invokes `go list -json` for the provided patterns and returns the package metadata.
func listPackages(patterns []string) ([]pkgInfo, error) {
	args := append([]string{"list", "-json"}, patterns...)
	cmd := exec.Command("go", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	defer cmd.Wait()

	dec := json.NewDecoder(bufio.NewReader(stdout))
	var pkgs []pkgInfo
	for dec.More() {
		var info pkgInfo
		if err := dec.Decode(&info); err != nil {
			return nil, err
		}
		pkgs = append(pkgs, info)
	}
	if err := cmd.Wait(); err != nil {
		return nil, err
	}
	return pkgs, nil
}

// isGeneratedFile checks if the file starts with the standard "Code generated" header.
func isGeneratedFile(filename string) bool {
	f, err := os.Open(filename)
	if err != nil {
		return false
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for i := 0; i < 10 && scanner.Scan(); i++ {
		line := scanner.Text()
		if strings.Contains(line, "Code generated") || strings.Contains(line, "DO NOT EDIT") {
			return true
		}
	}
	return false
}

// relativePath converts an absolute path to one relative to the repo root when possible.
func relativePath(path string) string {
	if rel, err := filepath.Rel(".", path); err == nil {
		return rel
	}
	return path
}

// shouldExclude verifica si una ruta de archivo relativa debe ser excluida basándose en listas de directorios y expresiones regulares.
func shouldExclude(rel string, dirs []string, regex []*regexp.Regexp) bool {
	for _, d := range dirs {
		if rel == d || strings.HasPrefix(rel, d+"/") {
			return true
		}
	}
	for _, rx := range regex {
		if rx.MatchString(rel) {
			return true
		}
	}
	return false
}
