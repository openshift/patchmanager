package classifiers

import (
	"github.com/openshift/patchmanager/pkg/config"
	"github.com/openshift/patchmanager/pkg/github"
)

// KeywordsClassifier classify pull request based on importance of bugzilla keywords.
type KeywordsClassifier struct {
	Config *config.KeywordsClassifierConfig
}

func (f *KeywordsClassifier) Score(pullRequest *github.PullRequest) float32 {
	highestScore := float32(0)
	for keyword, score := range *f.Config {
		for _, k := range pullRequest.Bug().Keywords {
			if k == keyword && score > highestScore {
				highestScore = score
			}
		}
	}
	return highestScore
}
