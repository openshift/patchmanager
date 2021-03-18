package v1

import "gopkg.in/yaml.v2"

type CandidateList struct {
	Items []Candidate `yaml:"items"`
}

type ApprovedCandidateList struct {
	Items []ApprovedCandidate `yaml:items`
}

type Candidate struct {
	yaml.CommentedMapSlice `yaml:"pullRequest"`

	Decision       string  `yaml:"-"`
	DecisionReason string  `yaml:"-"`
	PMScore        string  `yaml:"-"`
	Score          float32 `yaml:"-"`
	Description    string  `yaml:"-"`
	PullRequestURL string  `yaml:"-"`
	BugNumber      string  `yaml:"-"`
	Component      string  `yaml:"-"`
	Severity       string  `yaml:"-"`
}

type ApprovedCandidate struct {
	PullRequest PullRequest `yaml:"pullRequest"`
}

type PullRequest struct {
	URL            string  `yaml:"url"`
	Decision       string  `yaml:"decision"`
	DecisionReason string  `yaml:"decisionReason"`
	Score          float32 `yaml:"score"`
}
