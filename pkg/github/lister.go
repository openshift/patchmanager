package github

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/eparis/bugzilla"

	"github.com/google/go-github/v32/github"
	"golang.org/x/oauth2"
)

const queryTemplate = "org:kube-reporting org:openshift org:operator-framework label:lgtm label:approved label:bugzilla/valid-bug " +
	"base:release-%[1]s base:openshift-%[1]s base:enterprise-%[1]s is:open -repo:openshift/openshift-docs -label:do-not-merge/hold"

type PullRequestLister struct {
	ghClient *github.Client
	bzClient bugzilla.Client
}

func NewPullRequestLister(ctx context.Context, ghToken string, bzToken string) *PullRequestLister {
	return &PullRequestLister{
		bzClient: bugzilla.NewClient(func() []byte {
			return []byte(bzToken)
		}, "https://bugzilla.redhat.com"),
		ghClient: github.NewClient(oauth2.NewClient(ctx, oauth2.StaticTokenSource(&oauth2.Token{AccessToken: ghToken}))),
	}
}

func (l *PullRequestLister) ListForRelease(ctx context.Context, release, labels string) ([]*PullRequest, error) {
	query := buildGithubSearchQuery(queryTemplate, release) + " " + labels
	result, _, err := l.ghClient.Search.Issues(ctx, query, &github.SearchOptions{Sort: "updated", ListOptions: github.ListOptions{PerPage: 150}})
	if err != nil {
		return nil, err
	}
	var pullRequests []*PullRequest
	for i := range result.Issues {
		if !result.Issues[i].IsPullRequest() {
			continue
		}

		newPullRequest := &PullRequest{
			Issue: *result.Issues[i],
			Score: 0,
		}
		bugNumber := parseBugNumber(newPullRequest.Issue.GetTitle())

		if newPullRequest.bugID, err = strconv.Atoi(bugNumber); len(bugNumber) == 0 || err != nil {
			fmt.Printf("WARNING: Pull Request with invalid title: %s: %s (%v)\n", newPullRequest.Issue.GetHTMLURL(), newPullRequest.Issue.GetTitle(), err)
			continue
		}
		newPullRequest.getBugFn = func(id int) *bugzilla.Bug {
			bz, err := l.bzClient.GetBug(id)
			if err != nil {
				fmt.Printf("Failed to fetch bug #%d for %s: %s\n", id, newPullRequest.Issue.GetHTMLURL(), err)
				return nil
			}
			return bz
		}

		pullRequests = append(pullRequests, newPullRequest)
	}

	return pullRequests, nil
}

func (l *PullRequestLister) ListApprovedForRelease(ctx context.Context, release string) ([]*PullRequest, error) {
	return l.ListForRelease(ctx, release, "label:cherry-pick-approved")
}

func (l *PullRequestLister) ListCandidatesForRelease(ctx context.Context, release string) ([]*PullRequest, error) {
	return l.ListForRelease(ctx, release, "-label:cherry-pick-approved")
}

// parseBugNumber takes pull request title "Bug ####: Description" and return the ####
func parseBugNumber(pullRequestTitle string) string {
	re := regexp.MustCompile(`bug (\d+):`)
	matches := re.FindAllString(strings.ToLower(pullRequestTitle), 1)
	if len(matches) == 0 {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(strings.ToLower(strings.TrimSuffix(matches[0], ":")), "bug"))
}

func buildGithubSearchQuery(query, release string) string {
	return fmt.Sprintf(query, release)
}
