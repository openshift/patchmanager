package classifiers

import (
	"strconv"

	v1 "github.com/mfojtik/patchmanager/pkg/api/v1"
	"github.com/mfojtik/patchmanager/pkg/github"
)

// ProductManagementScoreClassifier classify pull request based on the product management score (PMScore).
type ProductManagementScoreClassifier struct {
	Config *v1.PMScoreClassifierConfig
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
