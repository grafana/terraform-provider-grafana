package main

import (
	"flag"
	"fmt"
	"go/format"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Find all references in the examples directory and in test files and generate a map of known references.
// Write that map in the specified file (in args).
func main() {
	// Parse flags
	var (
		walkDir     string
		fileToWrite string
	)
	flag.StringVar(&walkDir, "walk-dir", "", "directory to walk and find references in")
	flag.StringVar(&fileToWrite, "file", "", "file to write the known references to")
	flag.Parse()

	if walkDir == "" || fileToWrite == "" {
		log.Fatal("examples-dir and file flags are required")
	}

	walkDir, err := filepath.Abs(walkDir)
	if err != nil {
		log.Fatal(err)
	}

	exampleFiles := []string{}
	if err := filepath.Walk(walkDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if strings.HasPrefix(info.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) == ".tf" {
			exampleFiles = append(exampleFiles, path)
		}
		if strings.HasSuffix(path, "_test.go") {
			exampleFiles = append(exampleFiles, path)
		}
		return nil
	}); err != nil {
		log.Fatal(err)
	}

	knownReferencesMap := map[string]struct{}{}
	for _, file := range exampleFiles {
		log.Printf("Processing file: %s\n", file)

		bytes, err := os.ReadFile(file)
		if err != nil {
			log.Fatal(err)
		}

		for _, entry := range findReferences(string(bytes)) {
			knownReferencesMap[entry] = struct{}{}
		}
	}
	var knownReferences []string
	for k := range knownReferencesMap {
		knownReferences = append(knownReferences, k)
	}
	sort.Strings(knownReferences)

	// Write the known references to the specified file using os.Root
	// to scope filesystem operations to the file's directory.
	log.Printf("Writing known references to %s\n", fileToWrite)

	if err := writeKnownReferences(fileToWrite, knownReferences); err != nil {
		log.Fatal(err)
	}
}

var (
	resourceRe   = regexp.MustCompile(`resource\s+"(\w+)"\s+"([\w-]+)"\s+\{`)
	blockRe      = regexp.MustCompile(`^\s*(\w+)\s*\{\s*$`)
	assignmentRe = regexp.MustCompile(`\s*(\w+)\s*=\s*(?:\[\s*)?(\w+)\.(\w+)\.(\w+(?:\.\w+)*)`)
)

// findReferences returns the references found in HCL content, as `<resource>.<attribute path>=<resource>.<attribute path>`.
// Attributes in nested blocks are prefixed with their block path, e.g. `grafana_rule_group.rule.notification_settings.contact_point`.
func findReferences(content string) []string {
	var references []string
	var currentResource string
	// Braces opened since the resource line: the nested block name, or "" for anything else (maps, heredocs, ...).
	var blocks []string
	for _, line := range strings.Split(content, "\n") {
		trimmedLine := strings.TrimSpace(line)
		if strings.HasPrefix(trimmedLine, "output ") || strings.HasPrefix(trimmedLine, "data ") {
			currentResource = ""
			continue
		}

		resourceMatch := resourceRe.FindStringSubmatch(line)
		if resourceMatch != nil {
			currentResource = resourceMatch[1]
			blocks = nil
			continue
		}

		if currentResource == "" {
			continue
		}

		assignmentMatch := assignmentRe.FindStringSubmatch(line)
		if assignmentMatch != nil {
			refToResource, refToAttribute := assignmentMatch[2], assignmentMatch[4]
			if strings.HasPrefix(refToResource, "grafana_") { // TODO: Enable data resources
				attrPath := []string{currentResource}
				for _, block := range blocks {
					if block != "" {
						attrPath = append(attrPath, block)
					}
				}
				attrPath = append(attrPath, assignmentMatch[1])
				references = append(references, fmt.Sprintf("%s=%s.%s", strings.Join(attrPath, "."), refToResource, refToAttribute))
			}
		}

		opened := strings.Count(line, "{") - strings.Count(line, "}")
		for ; opened > 0; opened-- {
			block := ""
			if blockMatch := blockRe.FindStringSubmatch(line); blockMatch != nil {
				block = blockMatch[1]
			}
			blocks = append(blocks, block)
		}
		for ; opened < 0; opened++ {
			if len(blocks) == 0 {
				currentResource = "" // The resource's own closing brace
				break
			}
			blocks = blocks[:len(blocks)-1]
		}
	}
	return references
}

func writeKnownReferences(fileToWrite string, knownReferences []string) error {
	absFileToWrite, err := filepath.Abs(fileToWrite)
	if err != nil {
		return err
	}
	dir := filepath.Dir(absFileToWrite)
	base := filepath.Base(absFileToWrite)

	root, err := os.OpenRoot(dir)
	if err != nil {
		return err
	}
	defer root.Close()

	bytes, err := root.ReadFile(base)
	if err != nil {
		return err
	}
	stat, err := root.Stat(base)
	if err != nil {
		return err
	}

	content := string(bytes)
	start := strings.Index(content, "var knownReferences = []string{")
	if start == -1 {
		return fmt.Errorf("could not find knownReferences var in %s", fileToWrite)
	}
	end := strings.Index(content[start:], "}")

	knownReferencesStr := "var knownReferences = []string{\n"
	for _, v := range knownReferences {
		knownReferencesStr += fmt.Sprintf("%q,\n", v)
	}
	knownReferencesStr += "}"
	fmt.Println(knownReferencesStr)

	content = content[:start] + knownReferencesStr + content[start+end+1:]

	// Run gofmt on the content
	bytesToWrite, err := format.Source([]byte(content))
	if err != nil {
		return err
	}

	return root.WriteFile(base, bytesToWrite, stat.Mode())
}
