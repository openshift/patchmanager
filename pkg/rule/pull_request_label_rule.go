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
	matchesSeverity := len(p.Config.PermitSeverity) == 0  // If there are no entries, allow any severity
	for _, l := range pullRequest.Issue.Labels {
		for _, c := range p.Config.RefuseOnLabel {
			if strings.HasPrefix(l.GetName(), c) {
				result = append(result, fmt.Sprintf("skipping because %q label found", l.GetName()))
			}
		}
		if !matchesSeverity {
			for _, permitSeverity := range p.Config.PermitSeverity {
				if strings.HasSuffix(l.GetName(), fmt.Sprintf("severity-%s", permitSeverity) ) {
					matchesSeverity = true
					break
				}
			}
		}
	}

	if !matchesSeverity {
		result = append(result, fmt.Sprintf("skipping because only the following severities are being considered for this release: %v; close this PR or reassess the severity of the issue for this release", p.Config.PermitSeverity))
	}

	return result, len(result) == 0
}
