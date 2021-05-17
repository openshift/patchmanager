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
	return result, len(result) == 0
}
