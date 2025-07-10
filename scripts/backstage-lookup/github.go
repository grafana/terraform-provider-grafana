package main

import (
	"context"
	"fmt"
	"os"

	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
)

type GitHubClient struct {
	Client *githubv4.Client
}

func NewGitHubClient() (*GitHubClient, error) {
	accessToken := os.Getenv("GITHUB_TOKEN")
	if accessToken == "" {
		return nil, fmt.Errorf("GITHUB_TOKEN required")
	}
	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: accessToken},
	)
	ctx := context.Background()
	httpClient := oauth2.NewClient(ctx, src)

	return &GitHubClient{
		Client: githubv4.NewClient(httpClient),
	}, nil
}

func (g *GitHubClient) AddIssueToProject(org, repo string, issueNumber, projectNumber int) error {
	ctx := context.Background()

	contentId, err := g.findIssueId(ctx, org, repo, issueNumber)
	if err != nil {
		return err
	}

	projectId, err := g.findProjectId(ctx, org, projectNumber)
	if err != nil {
		return err
	}

	return g.addIssueToProject(ctx, contentId, projectId)
}

func (g *GitHubClient) RemoveIssueFromProject(org, repo string, issueNumber, projectNumber int) error {
	ctx := context.Background()

	itemId, projectId, err := g.findProjectItemId(ctx, org, repo, projectNumber, issueNumber)
	if err != nil {
		return err
	}

	if itemId == "" {
		return nil
	}

	return g.removeIssueFromProject(ctx, itemId, projectId)
}

func (g *GitHubClient) findIssueId(ctx context.Context, org, repo string, number int) (string, error) {
	var query struct {
		Repository struct {
			Issue struct {
				Id string
			} `graphql:"issue(number: $number)"`
		} `graphql:"repository(owner: $owner, name: $name)"`
	}

	err := g.Client.Query(ctx, &query, map[string]any{
		"owner":  githubv4.String(org),
		"name":   githubv4.String(repo),
		"number": githubv4.Int(number),
	})
	return query.Repository.Issue.Id, err
}

func (g *GitHubClient) findProjectItemId(ctx context.Context, org, repo string, projectNumber, issueNumber int) (string, string, error) {
	var query struct {
		Repository struct {
			Issue struct {
				ProjectItems struct {
					Edges []struct {
						Node struct {
							Id            string
							ProjectV2Item struct {
								Project struct {
									Id     string
									Number int
								}
							} `graphql:"... on ProjectV2Item"`
						}
					}
				} `graphql:"projectItems(first: 100)"` // assumes issues don't have more than 100 projects
			} `graphql:"issue(number: $number)"`
		} `graphql:"repository(owner: $owner, name: $name)"`
	}

	if err := g.Client.Query(ctx, &query, map[string]any{
		"owner":  githubv4.String(org),
		"name":   githubv4.String(repo),
		"number": githubv4.Int(issueNumber),
	}); err != nil {
		return "", "", err
	}

	for _, edge := range query.Repository.Issue.ProjectItems.Edges {
		if edge.Node.ProjectV2Item.Project.Number == projectNumber {
			return edge.Node.Id, edge.Node.ProjectV2Item.Project.Id, nil
		}
	}

	// not sure if we should return an error, not finding a project could be expected
	return "", "", nil //fmt.Errorf("findProjectItemId: issue not found on project")
}

func (g *GitHubClient) findProjectId(ctx context.Context, org string, number int) (string, error) {
	var query struct {
		Organization struct {
			ProjectV2 struct {
				Id string
			} `graphql:"projectV2(number: $number)"`
		} `graphql:"organization(login: $owner)"`
	}

	err := g.Client.Query(ctx, &query, map[string]any{
		"owner":  githubv4.String(org),
		"number": githubv4.Int(number),
	})
	return query.Organization.ProjectV2.Id, err
}

func (g *GitHubClient) addIssueToProject(ctx context.Context, contentId, projectId string) error {
	var mutation struct {
		AddProjectV2ItemById struct {
			ClientMutationId string
		} `graphql:"addProjectV2ItemById(input: $input)"`
	}
	input := githubv4.AddProjectV2ItemByIdInput{
		ContentID: githubv4.ID(contentId),
		ProjectID: githubv4.ID(projectId),
	}

	return g.Client.Mutate(ctx, &mutation, input, nil)
}

func (g *GitHubClient) removeIssueFromProject(ctx context.Context, itemId, projectId string) error {
	var mutation struct {
		DeleteProjectV2Item struct {
			ClientMutationId string
		} `graphql:"deleteProjectV2Item(input: $input)"`
	}

	input := githubv4.DeleteProjectV2ItemInput{
		ItemID:    githubv4.ID(itemId),
		ProjectID: githubv4.ID(projectId),
	}

	return g.Client.Mutate(ctx, &mutation, input, nil)
}
