package classifier

import (
	"strconv"
	"strings"

	v1 "github.com/mfojtik/patchmanager/pkg/api/v1"

	"github.com/mfojtik/patchmanager/pkg/github"
)

type Classifier interface {
	Score(*github.PullRequest) float32
}

type MultiClassifier struct {
	classifiers []Classifier
}

func (m *MultiClassifier) Score(pullRequest *github.PullRequest) float32 {
	score := float32(0)
	for i := range m.classifiers {
		score += m.classifiers[i].Score(pullRequest)
	}
	return score
}

func New(classifiers ...Classifier) Classifier {
	return &MultiClassifier{classifiers: classifiers}
}

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

// ComponentClassifier classify pull request based on bugzilla component.
// Some components are more critical to keep the platform on the wheels than others, these components should get more score.
type ComponentClassifier struct {
	Config *v1.ComponentClassifierConfig
}

func (c *ComponentClassifier) Score(pullRequest *github.PullRequest) float32 {
	score, ok := (*c.Config)[strings.ToLower(pullRequest.Bug().Component[0])]
	if !ok {
		return 0
	}
	return score
}

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
