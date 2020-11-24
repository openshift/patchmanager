package run

import (
	"context"
	"fmt"
	"os"
	"sort"

	"github.com/mfojtik/patchmanager/pkg/classifier"
	"github.com/mfojtik/patchmanager/pkg/github"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/klog/v2"
)

// runOptions holds values to drive the start command.
type runOptions struct {
	bugzillaAPIKey string
	githubToken    string
	release        string

	classifier classifier.Classifier
}

// NewRunCommand creates a render command.
func NewRunCommand(ctx context.Context) *cobra.Command {
	runOpts := runOptions{}
	cmd := &cobra.Command{
		Use: "run",
		Run: func(cmd *cobra.Command, args []string) {
			if err := runOpts.Complete(); err != nil {
				klog.Exit(err)
			}
			if err := runOpts.Validate(); err != nil {
				klog.Exit(err)
			}
			if err := runOpts.Run(ctx); err != nil {
				klog.Exit(err)
			}
		},
	}

	runOpts.AddFlags(cmd.Flags())

	return cmd
}

func (r *runOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&r.githubToken, "github-token", "", "Github Access Token (GITHUB_TOKEN env variable)")
	fs.StringVar(&r.bugzillaAPIKey, "bugzilla-apikey", "", "Bugzilla API Key (BUGZILLA_APIKEY env variable)")
	fs.StringVar(&r.release, "release", "", "Target release (eg. 4.6, 4.7, etc...)")
}

func (r *runOptions) Validate() error {
	if len(r.bugzillaAPIKey) == 0 {
		return fmt.Errorf("bugzilla-apikey flag must be specified or BUGZILLA_APIKEY environment must be set")
	}
	if len(r.githubToken) == 0 {
		return fmt.Errorf("github-token flag must be specified or GITHUB_TOKEN environment must be set")
	}
	if len(r.release) == 0 {
		return fmt.Errorf("release flag must be set")
	}
	return nil
}

func (r *runOptions) Complete() error {
	if len(r.bugzillaAPIKey) == 0 {
		r.bugzillaAPIKey = os.Getenv("BUGZILLA_APIKEY")
	}
	if len(r.githubToken) == 0 {
		r.githubToken = os.Getenv("GITHUB_TOKEN")
	}

	r.classifier = classifier.New(
		&classifier.SeverityClassifier{},
		&classifier.ComponentClassifier{},
		&classifier.FlagsClassifier{},
		&classifier.ProductManagementScoreClassifier{},
	)
	return nil
}

func (r *runOptions) Run(ctx context.Context) error {
	pendingPullRequests, err := github.NewPullRequestLister(ctx, r.githubToken, r.bugzillaAPIKey).ListForRelease(ctx, r.release)
	if err != nil {
		return err
	}

	klog.Infof("Processing %d cherry-pick pending pull requests ...", len(pendingPullRequests))

	for i := range pendingPullRequests {
		pendingPullRequests[i].Score = r.classifier.Score(pendingPullRequests[i])
	}

	sort.Slice(pendingPullRequests, func(i, j int) bool {
		return pendingPullRequests[i].Score > pendingPullRequests[j].Score
	})

	trim := func(in string) string {
		if len(in) > 150 {
			return in[0:150] + "... "
		}
		return in
	}

	for _, p := range pendingPullRequests {
		fmt.Printf("[%.1f][%-6s][%s] %s\n", p.Score, p.Bug().Severity, p.Issue.GetHTMLURL(), trim(p.Issue.GetTitle()))
	}

	return nil
}
