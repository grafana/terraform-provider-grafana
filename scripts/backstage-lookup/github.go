package main

import (
	"context"
	"os"

	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
)

type GitHubClient struct {
	Client *githubv4.Client
}

func NewGitHubClient() GitHubClient {
	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: os.Getenv("GITHUB_TOKEN")},
	)
	ctx := context.Background()
	httpClient := oauth2.NewClient(ctx, src)

	return GitHubClient{
		Client: githubv4.NewClient(httpClient),
	}
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
