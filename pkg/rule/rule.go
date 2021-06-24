package rule

import "github.com/openshift/patchmanager/pkg/github"

type Ruler interface {
	Evaluate(*github.PullRequest) ([]string, bool)
}

type MultiRuler struct {
	rulers []Ruler
}

func (m *MultiRuler) Evaluate(pullRequest *github.PullRequest) ([]string, bool) {
	decisions := []string{}
	result := true
	for i := range m.rulers {
		messages, pass := m.rulers[i].Evaluate(pullRequest)
		if !pass {
			decisions = append(decisions, messages...)
			result = false
		}
	}
	return decisions, result
}

func NewMultiRuler(rullers ...Ruler) Ruler {
	return &MultiRuler{rulers: rullers}
}
