package rule

import (
	"fmt"
	"strings"

	"github.com/openshift/patchmanager/pkg/config"
	"github.com/openshift/patchmanager/pkg/github"
)

type PullRequestLabelRule struct {
	Config *config.PullRequestLabelRuleConfig
}

func (p *PullRequestLabelRule) Evaluate(pullRequest *github.PullRequest) ([]string, bool) {
	result := []string{}
	for _, l := range pullRequest.Issue.Labels {
		for _, c := range p.Config.RefuseOnLabel {
			if strings.HasPrefix(l.GetName(), c) {
				result = append(result, fmt.Sprintf("skipping because %q label found", l.GetName()))
			}
		}
	}

	for _, c := range p.Config.RequireLabel {
		found := false
		for _, l := range pullRequest.Issue.Labels {
			if strings.HasPrefix(l.GetName(), c) {
				found = true
				break
			}
		}
		if found {
			continue
		}
		result = append(result, fmt.Sprintf("skipping because %q label was not found", c))
	}

	return result, len(result) == 0
}
