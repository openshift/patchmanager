package api

import (
	"fmt"
	"strings"

	v1 "github.com/mfojtik/patchmanager/pkg/api/v1"
	"gopkg.in/yaml.v2"
)

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
Score: %.2f
`, sanitizeSummary(candidates[i].Description), fmt.Sprintf("https://bugzilla.redhat.com/show_bug.cgi?id=%s", candidates[i].BugNumber), candidates[i].Component, candidates[i].Severity, candidates[i].PMScore, candidates[i].Score),
				},
				{
					MapItem: yaml.MapItem{Key: "decision", Value: candidates[i].Decision},
				},
			},
		}
	}
	return v1.CandidateList{items}
}

func sanitizeSummary(in string) string {
	return strings.ReplaceAll(strings.TrimSpace(in), "\n", " ")
}
