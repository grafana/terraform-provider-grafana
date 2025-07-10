package main

import (
	"fmt"
	"log"
	"os"
	"slices"
	"strconv"
	"strings"
)

func unique(slice []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(slice))
	for _, item := range slice {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}
	return result
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	if len(os.Args) < 3 {
		log.Fatal("Usage: backstage-lookup <issueNumber> <resource1> [resource2] ...")
	}

	backstage, err := NewBackstageClient()
	if err != nil {
		log.Fatal(err)
	}
	backstage.Filters = func(resourceName string) []string {
		return []string{
			fmt.Sprintf("kind=Component,metadata.name=resource-%s", resourceName),
			fmt.Sprintf("kind=Component,metadata.name=datasource-%s", resourceName),
		}
	}

	issueNumber, err := strconv.Atoi(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}

	var allProjects []string
	for _, resource := range os.Args[2:] {
		if resource = strings.TrimSpace(resource); resource != "" {
			projects, err := backstage.FindProjectsForResource(resource)
			if err != nil {
				log.Fatal(err)
			}
			allProjects = append(allProjects, projects...)
		}
	}

	allProjects = unique(allProjects)
	fmt.Printf("Assigning issue #%d to projects=%s\n", issueNumber, strings.Join(allProjects, " "))

	github, err := NewGitHubClient()
	if err != nil {
		log.Fatal(err)
	}

	// If the resource is not owned by monitoring and there are other projects claiming ownership, then remove monitoring.
	resourceIsOwnedByPlatformMonitoring := -1 != slices.IndexFunc(allProjects, func(p string) bool { return p == "513" })
	if len(allProjects) > 0 && !resourceIsOwnedByPlatformMonitoring {
		if err := github.RemoveIssueFromProject("grafana", "terraform-provider-grafana", issueNumber, 513); err != nil {
			log.Fatal(err)
		}
	}

	for _, projectNumber := range allProjects {
		projectNumberInt, err := strconv.Atoi(projectNumber)
		if err != nil {
			log.Fatal(err)
		}
		if err := github.AddIssueToProject("grafana", "terraform-provider-grafana", issueNumber, projectNumberInt); err != nil {
			log.Fatal(err)
		}
	}
}
