package classifiers

import (
	v1 "github.com/mfojtik/patchmanager/pkg/api/v1"
	"github.com/mfojtik/patchmanager/pkg/github"
)

// FlagsClassifier classify pull request based on importance of bugzilla flags.
type FlagsClassifier struct {
	Config *v1.FlagClassifierConfig
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
