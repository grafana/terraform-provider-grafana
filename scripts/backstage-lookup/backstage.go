package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/datolabs-io/go-backstage/v3"
	"github.com/mitchellh/mapstructure"
	"golang.org/x/oauth2"
)

type BackstageClient struct {
	Client  *backstage.Client
	Filters func(string) []string
}

func NewBackstageClient() (*BackstageClient, error) {
	baseURL := os.Getenv("BACKSTAGE_URL")
	if baseURL == "" {
		return nil, fmt.Errorf("BACKSTAGE_URL required")
	}

	accessToken := os.Getenv("BACKSTAGE_TOKEN")
	if accessToken == "" {
		return nil, fmt.Errorf("BACKSTAGE_TOKEN required")
	}

	src := oauth2.StaticTokenSource(
		&oauth2.Token{
			AccessToken: accessToken,
			TokenType:   "Bearer",
		})
	ctx := context.Background()
	httpClient := oauth2.NewClient(ctx, src)

	client, err := backstage.NewClient(baseURL, "default", httpClient)
	if err != nil {
		return nil, err
	}

	return &BackstageClient{
		Client: client,
		Filters: func(resourceName string) []string {
			return []string{
				fmt.Sprintf("kind=Component,metadata.name=%s", resourceName),
			}
		},
	}, nil
}

func (b *BackstageClient) FindProjectsForResource(resourceName, groupRef string) ([]string, error) {
	if groupRef == "" {
		resources, err := b.findComponents(resourceName)
		if err != nil {
			return nil, err
		}
		if len(resources) > 1 {
			log.Printf("Multiple components found, using first %s.", resources[0].Metadata.Name)
		}
		groupRef = resources[0].Spec.Owner
	}

	projects, err := b.findProjectsForGroup(groupRef)
	if err != nil {
		return nil, err
	}

	if len(projects) == 0 {
		return nil, fmt.Errorf("FindProjectForResource: no projects found")
	}

	// URL must look like https://github.com/orgs/<org>/projects/<number>
	re := regexp.MustCompile(`https://github.com/orgs/.*/projects/(\d+).*`)

	var ids []string
	for _, project := range projects {
		ids = append(ids, string(re.FindSubmatch([]byte(project))[1]))
	}
	return ids, nil
}

func (b *BackstageClient) findComponents(resourceName string) ([]backstage.ComponentEntityV1alpha1, error) {
	ctx := context.Background()
	entities, _, err := b.Client.Catalog.Entities.List(ctx, &backstage.ListEntityOptions{
		Filters: b.Filters(resourceName),
	})
	if err != nil {
		return nil, err
	}
	if len(entities) == 0 {
		return nil, fmt.Errorf("findComponents: No entities found.")
	}
	if len(entities) > 1 {
		log.Printf("Multiple entities found.")
	}

	components := make([]backstage.ComponentEntityV1alpha1, len(entities))
	if err := mapstructure.Decode(entities, &components); err != nil {
		return nil, err
	}

	return components, nil
}

func (b *BackstageClient) findGroupByRef(ref string) (*backstage.GroupEntityV1alpha1, error) {
	entityRef, err := parseEntityRef(ref)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	entities, _, err := b.Client.Catalog.Entities.List(ctx, &backstage.ListEntityOptions{
		Filters: []string{
			fmt.Sprintf("kind=Group,metadata.name=%s,metadata.namespace=%s", entityRef.Name, entityRef.Namespace),
		},
	})
	if err != nil {
		return nil, err
	}
	if len(entities) == 0 {
		return nil, fmt.Errorf("findGroupByRef: No entities found.")
	}
	if len(entities) > 1 {
		return nil, fmt.Errorf("findGroupByRef: Multiple entities found.")
	}
	var group backstage.GroupEntityV1alpha1
	if err := mapstructure.Decode(entities[0], &group); err != nil {
		return nil, err
	}

	group.Metadata = entities[0].Metadata
	group.Relations = entities[0].Relations

	return &group, nil
}

func (b *BackstageClient) findProjectsForGroup(groupRef string) ([]string, error) {
	group, err := b.findGroupByRef(groupRef)
	if err != nil {
		return nil, err
	}

	var githubProjects []string
	for _, link := range group.Metadata.Links {
		if link.Type == "github_project" {
			githubProjects = append(githubProjects, link.URL)
		}
	}
	if len(githubProjects) == 0 {
		for _, relation := range group.Relations {
			if relation.Type == "parentOf" {
				projects, _ := b.findProjectsForGroup(relation.TargetRef)
				githubProjects = append(githubProjects, projects...)
			}
		}
	}
	return githubProjects, nil
}

type EntityRef struct {
	Kind      string
	Namespace string
	Name      string
}

func parseEntityRef(ref string) (*EntityRef, error) {
	kindParts := strings.Split(ref, ":")
	if len(kindParts) != 2 {
		return nil, fmt.Errorf("Could not parse entityRef.")
	}

	parts := strings.Split(kindParts[1], "/")
	if len(parts) != 2 {
		return nil, fmt.Errorf("Could not parse entityRef.")
	}
	return &EntityRef{
		Kind:      kindParts[0],
		Namespace: parts[0],
		Name:      parts[1],
	}, nil
}
