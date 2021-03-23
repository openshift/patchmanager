package classifiers

import (
	"strconv"

	"github.com/openshift/patchmanager/pkg/config"

	"github.com/openshift/patchmanager/pkg/github"
)

// ProductManagementScoreClassifier classify pull request based on the product management score (PMScore).
type ProductManagementScoreClassifier struct {
	Config *config.PMScoreClassifierConfig
}

func (p *ProductManagementScoreClassifier) Score(pullRequest *github.PullRequest) float32 {
	pmScore, err := strconv.Atoi(pullRequest.Bug().PMScore)
	if err != nil {
		return 0.0
	}
	for _, v := range *p.Config {
		if pmScore >= v.From && pmScore <= v.To {
			return v.Score
		}
	}
	return 0
}
