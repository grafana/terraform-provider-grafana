package main

import (
	"flag"
	"fmt"
	"log"
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

	dryRun := flag.Bool("dry-run", true, "Dry-run prints the intended actions.")
	groupRef := flag.String("group-ref", "", "Assign to project for this Backstage groupRef.")

	flag.Parse()

	if *dryRun {
		fmt.Println("Dry-run: use --dry-run=false to turn off")
	}

	positionalArgs := flag.Args()

	if len(positionalArgs) < 2 {
		log.Fatal("Usage: backstage-lookup <issueNumber> <resource1> [resource2] ...")
	}

	issueNumber, err := strconv.Atoi(positionalArgs[0])
	if err != nil {
		log.Fatal(err)
	}

	do(issueNumber, positionalArgs[1:], *dryRun, *groupRef)
}

func do(issueNumber int, resources []string, dryRun bool, groupRef string) {
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

	var allProjects []string
	for _, resource := range resources {
		if resource = strings.TrimSpace(resource); resource != "" {
			fmt.Printf("Looking up resource: %s\n", resource)
			projects, err := backstage.FindProjectsForResource(resource, groupRef)
			if err != nil {
				log.Printf("Warning: failed to find projects for resource %s: %v", resource, err)
				continue
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
		fmt.Printf("Removing issue #%d from platform-monitoring project (513)\n", issueNumber)
		if !dryRun {
			if err := github.RemoveIssueFromProject("grafana", "terraform-provider-grafana", issueNumber, 513); err != nil {
				log.Printf("Warning: failed to remove from platform-monitoring project: %v", err)
			}
		}
	}

	for _, projectNumber := range allProjects {
		projectNumberInt, err := strconv.Atoi(projectNumber)
		if err != nil {
			log.Printf("Warning: invalid project number %s: %v", projectNumber, err)
			continue
		}
		fmt.Printf("Adding issue #%d to project %d\n", issueNumber, projectNumberInt)
		if !dryRun {
			if err := github.AddIssueToProject("grafana", "terraform-provider-grafana", issueNumber, projectNumberInt); err != nil {
				log.Printf("Warning: failed to add to project %d: %v", projectNumberInt, err)
			}
		}
	}
}
