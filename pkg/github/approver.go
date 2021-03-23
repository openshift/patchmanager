package github

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/oauth2"

	"github.com/google/go-github/v32/github"
)

type PullRequestApprover struct {
	client *github.Client
}

func NewPullRequestApprover(ctx context.Context, ghToken string) *PullRequestApprover {
	return &PullRequestApprover{client: github.NewClient(oauth2.NewClient(ctx, oauth2.StaticTokenSource(&oauth2.Token{AccessToken: ghToken})))}
}

func (p *PullRequestApprover) CherryPickApprove(ctx context.Context, url string) error {
	owner, repo, number, err := parsePullRequestMeta(url)
	if err != nil {
		return err
	}
	_, _, err = p.client.Issues.AddLabelsToIssue(ctx, owner, repo, number, []string{"cherry-pick-approved"})
	return err
}

func (p *PullRequestApprover) Comment(ctx context.Context, url, comment string) error {
	owner, repo, number, err := parsePullRequestMeta(url)
	if err != nil {
		return err
	}
	_, _, err = p.client.Issues.CreateComment(ctx, owner, repo, number, &github.IssueComment{
		Body: &comment,
	})
	return err
}

func parsePullRequestMeta(u string) (string, string, int, error) {
	parts := strings.Split(strings.TrimPrefix(u, "https://github.com/"), "/")
	if len(parts) != 3 {
		return "", "", 0, fmt.Errorf("unable to parse pull request url %q", u)
	}
	number, err := strconv.Atoi(parts[2])
	if err != nil {
		return "", "", 0, err
	}
	return parts[0], parts[1], number, nil
}
