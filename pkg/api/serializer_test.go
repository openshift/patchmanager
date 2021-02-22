package api

import (
	"testing"

	"gopkg.in/yaml.v2"

	v1 "github.com/mfojtik/patchmanager/pkg/api/v1"
)

func TestMarshal(t *testing.T) {
	c := []v1.Candidate{
		{
			Score:          1,
			Description:    "Test1",
			Severity:       "urgent",
			PMScore:        "150",
			Component:      "Networking",
			BugNumber:      "1234566",
			PullRequestURL: "https://github.com/openshift/origin/pulls/1",
		},
		{
			Score:          2,
			Description:    "Test2",
			PullRequestURL: "https://github.com/openshift/origin/pulls/2",
		},
	}
	out, _ := yaml.Marshal(NewCandidateList(c))
	t.Logf("out: %s", string(out))
}
