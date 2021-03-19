package classifiers

import (
	"strings"

	v1 "github.com/openshift/patchmanager/pkg/api/v1"
	"github.com/openshift/patchmanager/pkg/github"
)

// SeverityClassifier classify pull request based on the bugzilla severity.
// Urgent:1, High:0.5, Medium:0.2, Low: 0.1
// Unknown severity gets penalty of -1.
type SeverityClassifier struct {
	Config *v1.SeverityClassifierConfig
}

func (s *SeverityClassifier) Score(pullRequest *github.PullRequest) float32 {
	score, ok := (*s.Config)[strings.ToLower(pullRequest.Bug().Severity)]
	if !ok {
		return 0
	}
	return score
}
