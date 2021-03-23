package classifiers

import (
	"github.com/openshift/patchmanager/pkg/config"
	"github.com/openshift/patchmanager/pkg/github"
)

// FlagsClassifier classify pull request based on importance of bugzilla flags.
type FlagsClassifier struct {
	Config *config.FlagClassifierConfig
}

func (f *FlagsClassifier) Score(pullRequest *github.PullRequest) float32 {
	highestScore := float32(0)
	for flag, score := range *f.Config {
		for _, f := range pullRequest.Bug().Flags {
			if f.Name == flag && score > highestScore {
				highestScore = score
			}
		}
	}
	return highestScore
}
