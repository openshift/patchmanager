package classifier

import (
	"strconv"
	"strings"

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
type SeverityClassifier struct{}

func (p *SeverityClassifier) Score(pullRequest *github.PullRequest) float32 {
	switch pullRequest.Bug().Severity {
	case "urgent":
		return 1
	case "high":
		return 0.5
	case "medium":
		return 0.2
	case "low":
		return 0.1
	case "unknown":
		return -1
	default:
		return 0
	}
}

// ComponentClassifier classify pull request based on bugzilla component.
// Some components are more critical to keep the platform on the wheels than others, these components should get more score.
type ComponentClassifier struct{}

func (c *ComponentClassifier) Score(pullRequest *github.PullRequest) float32 {
	switch strings.ToLower(pullRequest.Bug().Component[0]) {
	case "authentication", "networking", "node", "kube-apiserver":
		return 0.5
	default:
		return 0
	}
}

// FlagsClassifier classify pull request based on importance of bugzilla flags.
type FlagsClassifier struct{}

func (f *FlagsClassifier) Score(pullRequest *github.PullRequest) float32 {
	for _, f := range pullRequest.Bug().Flags {
		switch f.Name {
		case "TestBlocker":
			return 0.9
		case "UpgradeBlocker":
			return 1
		default:
			continue
		}
	}
	return 0
}

// ProductManagementScoreClassifier classify pull request based on the product management score (PMScore).
type ProductManagementScoreClassifier struct{}

func (p *ProductManagementScoreClassifier) Score(pullRequest *github.PullRequest) float32 {
	pmScore, err := strconv.Atoi(pullRequest.Bug().PMScore)
	if err != nil {
		return 0
	}
	switch {
	case pmScore >= 100:
		return 0.7
	case pmScore >= 50:
		return 0.5
	case pmScore < 50 && pmScore > 30:
		return 0.2
	default:
		return 0
	}
}
