package main

import (
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// formatOutput generates the final CODEOWNERS file content.
// staticContent is the raw content of CODEOWNERS.in, inserted between the
// default rule and the generated rules.
func formatOutput(defaultOwner string, staticContent string, rules []rule) string {
	var buf strings.Builder

	buf.WriteString("# Auto-generated — do not edit manually.\n")
	buf.WriteString("# For static rules, edit .github/CODEOWNERS.in instead.\n")
	buf.WriteString("# Regenerate with: make codeowners\n")
	buf.WriteString("\n")
	buf.WriteString(fmt.Sprintf("* @grafana/%s\n", defaultOwner))

	// Insert static rules from CODEOWNERS.in
	if staticContent != "" {
		buf.WriteString("\n")
		buf.WriteString(staticContent)
		if !strings.HasSuffix(staticContent, "\n") {
			buf.WriteString("\n")
		}
	}

	// Find the maximum pattern length for alignment
	maxLen := 0
	for _, r := range rules {
		if len(r.Pattern) > maxLen {
			maxLen = len(r.Pattern)
		}
	}

	// Group rules by package directory for section comments
	var currentPkg string
	for _, r := range rules {
		pkg := extractSectionFromPattern(r.Pattern)
		if pkg != currentPkg {
			currentPkg = pkg
			buf.WriteString("\n")
			buf.WriteString(fmt.Sprintf("# %s\n", pkg))
		}
		padding := maxLen - len(r.Pattern) + 3
		if padding < 1 {
			padding = 1
		}
		buf.WriteString(fmt.Sprintf("%s%s%s\n", r.Pattern, strings.Repeat(" ", padding), r.Team))
	}

	return buf.String()
}

// extractSectionFromPattern extracts a section label from a CODEOWNERS pattern.
// Used to group rules under section comments in the output.
func extractSectionFromPattern(pattern string) string {
	// internal/resources/<pkg>/... → "internal/resources/<pkg>"
	const resourcesPrefix = "/internal/resources/"
	if strings.HasPrefix(pattern, resourcesPrefix) {
		rest := strings.TrimPrefix(pattern, resourcesPrefix)
		if i := strings.Index(rest, "/"); i != -1 {
			return "internal/resources/" + rest[:i]
		}
		return pattern
	}

	// /examples/resources/... → "examples/resources"
	// /examples/data-sources/... → "examples/data-sources"
	// /docs/resources/... → "docs/resources"
	// /docs/data-sources/... → "docs/data-sources"
	for _, prefix := range []string{"/examples/resources/", "/examples/data-sources/", "/docs/resources/", "/docs/data-sources/"} {
		if strings.HasPrefix(pattern, prefix) {
			return strings.TrimPrefix(prefix, "/")[:len(prefix)-2] // strip leading "/" and trailing "/"
		}
	}

	return pattern
}

// parseStaticPatterns extracts CODEOWNERS patterns from static content.
// It ignores comments and blank lines, and takes the first whitespace-delimited
// token from each line as the pattern.
func parseStaticPatterns(content string) []string {
	var patterns []string
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 1 {
			patterns = append(patterns, fields[0])
		}
	}
	return patterns
}

// gitLsFiles returns all tracked files in the repository, relative to root.
func gitLsFiles(root string) ([]string, error) {
	cmd := exec.Command("git", "ls-files")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git ls-files: %w", err)
	}
	var files []string
	for _, line := range strings.Split(strings.TrimRight(string(out), "\n"), "\n") {
		if line != "" {
			files = append(files, line)
		}
	}
	return files, nil
}

// reportDefaultCoverage lists tracked files that only match the default
// CODEOWNERS rule (i.e., not covered by any package-specific pattern).
func reportDefaultCoverage(w io.Writer, root string, staticContent string, rules []rule) {
	patterns := make([]string, 0, len(rules))
	for _, r := range rules {
		patterns = append(patterns, r.Pattern)
	}
	patterns = append(patterns, parseStaticPatterns(staticContent)...)

	files, err := gitLsFiles(root)
	if err != nil {
		fmt.Fprintf(w, "WARNING: %v\n", err)
		return
	}

	var defaultFiles []string

	for _, rel := range files {
		if !matchesAnyRule(rel, patterns) {
			defaultFiles = append(defaultFiles, rel)
		}
	}

	fmt.Fprintf(w, "\nFiles covered by default rule (* @grafana/%s):\n", "platform-monitoring")
	dirCounts := make(map[string]int)
	var dirOrder []string
	for _, f := range defaultFiles {
		dir := filepath.Dir(f)
		if _, exists := dirCounts[dir]; !exists {
			dirOrder = append(dirOrder, dir)
		}
		dirCounts[dir]++
	}
	sort.Strings(dirOrder)

	for _, dir := range dirOrder {
		count := dirCounts[dir]
		if count == 1 {
			for _, f := range defaultFiles {
				if filepath.Dir(f) == dir {
					fmt.Fprintf(w, "  %s\n", f)
					break
				}
			}
		} else {
			fmt.Fprintf(w, "  %s/ (%d files)\n", dir, count)
		}
	}

}

// matchesAnyRule checks if a file path matches any of the CODEOWNERS patterns.
func matchesAnyRule(relPath string, patterns []string) bool {
	for _, pattern := range patterns {
		if matchPattern(relPath, pattern) {
			return true
		}
	}
	return false
}

// matchPattern matches a relative file path against a CODEOWNERS pattern.
// Supports single `*` (matches within one directory) and `**` (matches across directories).
func matchPattern(relPath string, pattern string) bool {
	pattern = strings.TrimPrefix(pattern, "/")

	// Handle /** (double-star) patterns: match anything under the directory
	if strings.HasSuffix(pattern, "/**") {
		dir := strings.TrimSuffix(pattern, "/**")
		return strings.HasPrefix(relPath, dir+"/")
	}

	patDir := filepath.Dir(pattern)
	patFile := filepath.Base(pattern)
	fileDir := filepath.Dir(relPath)
	fileName := filepath.Base(relPath)

	if patDir != fileDir {
		return false
	}

	matched, _ := filepath.Match(patFile, fileName)
	return matched
}

// diffLines prints a simple line-by-line diff between two strings.
func diffLines(w io.Writer, expected, actual string) {
	expectedLines := strings.Split(expected, "\n")
	actualLines := strings.Split(actual, "\n")

	maxLines := len(expectedLines)
	if len(actualLines) > maxLines {
		maxLines = len(actualLines)
	}

	for i := 0; i < maxLines; i++ {
		var eLine, aLine string
		if i < len(expectedLines) {
			eLine = expectedLines[i]
		}
		if i < len(actualLines) {
			aLine = actualLines[i]
		}
		if eLine != aLine {
			if i < len(expectedLines) {
				fmt.Fprintf(w, "- %s\n", eLine)
			}
			if i < len(actualLines) {
				fmt.Fprintf(w, "+ %s\n", aLine)
			}
		}
	}
}
