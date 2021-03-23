package classifiers

import (
	"strings"

	"github.com/openshift/patchmanager/pkg/config"

	"github.com/openshift/patchmanager/pkg/github"
)

// ComponentClassifier classify pull request based on bugzilla component.
// Some components are more critical to keep the platform on the wheels than others, these components should get more score.
type ComponentClassifier struct {
	Config *config.ComponentClassifierConfig
}

func (c *ComponentClassifier) Score(pullRequest *github.PullRequest) float32 {
	score, ok := (*c.Config)[strings.ToLower(pullRequest.Bug().Component[0])]
	if !ok {
		return 0
	}
	return score
}
