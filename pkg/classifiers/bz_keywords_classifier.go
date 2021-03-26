package classifiers

import (
	"github.com/openshift/patchmanager/pkg/config"
	"github.com/openshift/patchmanager/pkg/github"
)

// KeywordsClassifier classifies pull requests based on the importance of bugzilla keywords.
type KeywordsClassifier struct {
	Config *config.KeywordClassifierConfig
}

func (f *KeywordsClassifier) Score(pullRequest *github.PullRequest) float32 {
	highestScore := float32(0)
	for keyword, score := range *f.Config {
		for _, f := range pullRequest.Bug().Keywords {
			if f == keyword && score > highestScore {
				highestScore = score
			}
		}
	}
	return highestScore
}
