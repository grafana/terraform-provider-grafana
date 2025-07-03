package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

// Constants for API and configuration
const (
	defaultTimeout = 30 * time.Second
	groupPrefix    = "group:"
)

// Environment variable names
const (
	EnvBackstageURL = "BACKSTAGE_URL"
	EnvToken        = "TERRAFORM_AUTOMATION_TOKEN"
)

// API response types
type Component struct {
	Spec struct {
		Owner string `json:"owner"`
	} `json:"spec"`
}

type Group struct {
	Metadata struct {
		Links []struct {
			Type string `json:"type"`
			URL  string `json:"url"`
		} `json:"links"`
	} `json:"metadata"`

	Relations []struct {
		Type      string `json:"type"`
		TargetRef string `json:"targetRef"`
	} `json:"relations"`
}

// Result represents the lookup result
type Result struct {
	Projects []string
	Teams    []string
}

// BackstageLookup handles API interactions with Backstage
type BackstageLookup struct {
	client       *http.Client
	baseURL      string
	token        string
	projectRegex *regexp.Regexp
}

// NewBackstageLookup creates a new Backstage API client
func NewBackstageLookup(baseURL, token string) *BackstageLookup {
	return &BackstageLookup{
		client:       &http.Client{Timeout: defaultTimeout},
		baseURL:      baseURL,
		token:        token,
		projectRegex: regexp.MustCompile(`/projects/(\d+)`),
	}
}

// get performs an authenticated HTTP request to Backstage API
func (b *BackstageLookup) get(url string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+b.token)
	req.Header.Set("Accept", "application/json")

	resp, err := b.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

// findOwner retrieves the owner of a component by trying different prefixes and namespaces
func (b *BackstageLookup) findOwner(resource string) string {
	endpoints := []string{
		"default/resource-", "default/datasource-",
		"backstage-catalog/resource-", "backstage-catalog/datasource-",
	}

	for _, endpoint := range endpoints {
		url := fmt.Sprintf("%s/api/catalog/entities/by-name/component/%s%s", b.baseURL, endpoint, resource)

		data, err := b.get(url)
		if err != nil {
			continue
		}

		var comp Component
		if json.Unmarshal(data, &comp) == nil && comp.Spec.Owner != "" {
			return comp.Spec.Owner
		}
	}
	return ""
}

// findProject retrieves GitHub project information for a group
func (b *BackstageLookup) findProject(namespace, team string) string {
	url := fmt.Sprintf("%s/api/catalog/entities/by-name/group/%s/%s", b.baseURL, namespace, team)

	data, err := b.get(url)
	if err != nil {
		return ""
	}

	var group Group
	if json.Unmarshal(data, &group) != nil {
		return ""
	}

	for _, link := range group.Metadata.Links {
		if link.Type == "github_project" {
			if matches := b.projectRegex.FindStringSubmatch(link.URL); len(matches) >= 2 {
				return matches[1]
			}
		}
	}

	// Walk through parentOf relations and return first match
	// Background: the teams in group:default/ are synced from GitHub, we can't add arbitrary links to these, we have added the links to teams in group:backstage-catalog: instead. The teams in the backstage-catalog namespace refer to the GitHub teams as their parent. In theory multiple children could have a GitHub project, this loop returns the first match.
	for _, relation := range group.Relations {
		if relation.Type == "parentOf" {
			fmt.Println(relation.TargetRef)
			namespace, team := parseOwner(relation.TargetRef)
			project := b.findProject(namespace, team)
			if project != "" {
				return project
			}
		}
	}

	return ""
}

// parseOwner parses a group owner string into namespace and name
func parseOwner(owner string) (namespace, team string) {
	if !strings.HasPrefix(owner, groupPrefix) {
		return "", ""
	}

	parts := strings.Split(strings.TrimPrefix(owner, groupPrefix), "/")
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "", ""
}

// LookupResource looks up project and team information for a Terraform resource
func (b *BackstageLookup) LookupResource(resource string) (projects, teams []string) {
	if resource == "Other (please describe in the issue)" {
		return nil, nil
	}

	log.Printf("Processing: %s", resource)

	owner := b.findOwner(resource)
	if owner == "" {
		log.Printf("No owner found for %s - manual triage needed", resource)
		return nil, nil
	}

	namespace, team := parseOwner(owner)
	if namespace == "" || team == "" {
		log.Printf("Invalid owner %s for %s - manual triage needed", owner, resource)
		return nil, nil
	}

	log.Printf("Found owner %s for %s", owner, resource)

	if project := b.findProject(namespace, team); project != "" {
		log.Printf("Found project %s for team %s", project, team)
		return []string{project}, []string{team}
	}

	log.Printf("No project found for team %s", team)
	return nil, []string{team}
}

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

	if len(os.Args) < 2 {
		log.Fatal("Usage: backstage-lookup <resource1> [resource2] ...")
	}

	baseURL := os.Getenv("BACKSTAGE_URL")
	if baseURL == "" {
		log.Fatal("BACKSTAGE_URL required")
	}

	audience := os.Getenv("AUDIENCE")
	if audience == "" {
		log.Fatal("AUDIENCE required")
	}

	accessToken := os.Getenv("ACCESS_TOKEN")
	if accessToken == "" {
		log.Fatal("ACCESS_TOKEN required")
	}

	lookup := NewBackstageLookup(baseURL, accessToken)

	var allProjects, allTeams []string
	for _, resource := range os.Args[1:] {
		if resource = strings.TrimSpace(resource); resource != "" {
			// Clean resource name
			resource = strings.TrimSuffix(strings.TrimSuffix(resource, " (resource)"), " (data source)")
			projects, teams := lookup.LookupResource(resource)
			allProjects = append(allProjects, projects...)
			allTeams = append(allTeams, teams...)
		}
	}

	fmt.Printf("projects=%s\n", strings.Join(unique(allProjects), " "))
	fmt.Printf("teams=%s\n", strings.Join(unique(allTeams), " "))
}
