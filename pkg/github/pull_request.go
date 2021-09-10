package github

import (
	"github.com/eparis/bugzilla"
	"github.com/google/go-github/v32/github"
)

type PullRequest struct {
	Issue *github.Issue
	Score float32

	// do lazy fetch for bugs when needed to speed up sorting
	getBugFn func(int) *bugzilla.Bug
	bugID    int
	bug      *bugzilla.Bug
}

func (p *PullRequest) Bug() *bugzilla.Bug {
	if p.bug == nil {
		p.bug = p.getBugFn(p.bugID)
	}
	return p.bug
}
