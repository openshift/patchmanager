package api

import (
	"fmt"
	"strings"

	v1 "github.com/openshift/patchmanager/pkg/api/v1"
	"gopkg.in/yaml.v2"
)

// NewCandidateList takes list of candidates and transforms it to CandidateList which is serialized to YAML.
// YAML file include comments with additional information about pulls.
func NewCandidateList(candidates []v1.Candidate) v1.CandidateList {
	items := make([]v1.Candidate, len(candidates))

	for i := range candidates {
		items[i] = v1.Candidate{
			CommentedMapSlice: yaml.CommentedMapSlice{
				{
					MapItem: yaml.MapItem{Key: "url", Value: candidates[i].PullRequestURL},
					Comment: fmt.Sprintf(`Description: %s
Bug: %s
Component: %s
Severity: %s
PM Score %s
`, sanitizeSummary(candidates[i].Description), fmt.Sprintf("https://bugzilla.redhat.com/show_bug.cgi?id=%s", candidates[i].BugNumber), candidates[i].Component, candidates[i].Severity, candidates[i].PMScore),
				},
				{
					MapItem: yaml.MapItem{Key: "decision", Value: candidates[i].Decision},
				},
				{
					MapItem: yaml.MapItem{Key: "score", Value: candidates[i].Score},
				},
			},
		}
		if len(candidates[i].DecisionReason) > 0 {
			items[i].CommentedMapSlice = append(items[i].CommentedMapSlice, yaml.CommentedMapItem{
				MapItem: yaml.MapItem{Key: "decisionReason", Value: candidates[i].DecisionReason},
			})
		}
	}
	return v1.CandidateList{items}
}

func sanitizeSummary(in string) string {
	return strings.ReplaceAll(strings.TrimSpace(in), "\n", " ")
}
