package generate

import (
	"os"
	"sort"
	"strings"
)

func sortResourcesFile(filePath string) error {
	// Read the file
	content, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	stat, err := os.Stat(filePath)
	if err != nil {
		return err
	}

	// Rewrite the file with sorted resources
	content = []byte(sortResources(string(content)))
	return os.WriteFile(filePath, content, stat.Mode())
}

func sortResources(content string) string {
	spaceAtEnd := content[strings.LastIndex(content, "}")+1:]
	content = content[:strings.LastIndex(content, "}")+1]

	// Sort the generated file
	split := strings.Split(content, "\n\n")

	if len(split) < 2 {
		return content
	}

	index := 0
	for index < len(split) && !strings.Contains(split[index], `resource "`) {
		index++
	}

	content = strings.Join(split[:index], "\n\n")
	split = split[index:]

	sort.Slice(split, func(i, j int) bool {
		if !strings.Contains(split[i], `resource "`) {
			return true
		}
		if !strings.Contains(split[j], `resource "`) {
			return false
		}

		resourceName := func(text string) string { return strings.Split(text, "resource \"")[1] }
		return resourceName(split[i]) < resourceName(split[j])
	})

	if content != "" {
		split = append([]string{content}, split...)
	}
	return strings.Join(split, "\n\n") + spaceAtEnd
}
