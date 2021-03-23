package v1

import "gopkg.in/yaml.v2"

// CandidateList represets a list of candidates to approve
type CandidateList struct {
	Items []Candidate `yaml:"items"`
}

// Candidate represents a single pull request that is candidate for approval.
// This type contain pullRequest field that is used to describe the candidate metadata (bug, severity, etc)
// as YAML comments.
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

// ApprovedCandidateList represents a list of approved candidates
// This is used for parsing candidate list YAML, ignoring YAML comments.
type ApprovedCandidateList struct {
	Items []ApprovedCandidate `yaml:items`
}

type ApprovedCandidate struct {
	PullRequest ApprovedPullRequest `yaml:"pullRequest"`
}

type ApprovedPullRequest struct {
	URL            string  `yaml:"url"`
	Decision       string  `yaml:"decision"`
	DecisionReason string  `yaml:"decisionReason"`
	Score          float32 `yaml:"score"`
}
