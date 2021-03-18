package classifiers

import (
	"github.com/mfojtik/patchmanager/pkg/github"
)

// Classifier interface define Score function that every classifier must implement
type Classifier interface {
	Score(*github.PullRequest) float32
}

// MultiClassifier groups multiple classifier together and perform synchronous classifications
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

func NewMultiClassifier(classifiers ...Classifier) Classifier {
	return &MultiClassifier{classifiers: classifiers}
}
