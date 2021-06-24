package cleanup

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/openshift/patchmanager/pkg/cmd/util"

	"github.com/openshift/patchmanager/pkg/config"

	githubapi "github.com/google/go-github/v32/github"

	"github.com/dustin/go-humanize"
	"github.com/lensesio/tableprinter"
	"github.com/openshift/patchmanager/pkg/github"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/klog/v2"
)

// cleanupOptions holds values to drive the start command.
type cleanupOptions struct {
	release        string
	githubToken    string
	bugzillaAPIKey string
	config         *config.PatchManagerConfig
	configFile     string
}

// NewCleanupCommand creates a render command.
func NewCleanupCommand(ctx context.Context) *cobra.Command {
	runOpts := cleanupOptions{}
	cmd := &cobra.Command{
		Use:   "cleanup",
		Short: "Clean up cherry-pick-approved labels from approved pull requests",
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

func (r *cleanupOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&r.githubToken, "github-token", "", "Github Access Token (GITHUB_TOKEN env variable)")
	fs.StringVar(&r.bugzillaAPIKey, "bugzilla-apikey", "", "Bugzilla API Key (BUGZILLA_APIKEY env variable)")
	fs.StringVar(&r.release, "release", "", "Release to use to list candidates")
	fs.StringVar(&r.configFile, "config", os.Getenv("PATCHMANAGER_CONFIG"), "Path to a config file (PATCHMANAGER_CONFIG env variable)")
}

func (r *cleanupOptions) Validate() error {
	return nil
}

func (r *cleanupOptions) Complete() error {
	if len(r.bugzillaAPIKey) == 0 {
		r.bugzillaAPIKey = os.Getenv("BUGZILLA_APIKEY")
	}
	var err error
	r.config, err = config.GetConfig(r.configFile)
	if err != nil {
		return fmt.Errorf("unable to get config file %q: %v", r.configFile, err)
	}
	if len(r.githubToken) == 0 {
		r.githubToken = os.Getenv("GITHUB_TOKEN")
	}
	if len(r.config.Release) > 0 && len(r.release) == 0 {
		r.release = r.config.Release
	}
	return nil
}

type pull struct {
	URL        string `header:"URL"`
	LastUpdate string `header:"Last Update"`
}

func (r *cleanupOptions) Run(ctx context.Context) error {
	approved, err := github.NewPullRequestLister(ctx, r.githubToken, r.bugzillaAPIKey).ListApprovedForRelease(ctx, r.release)
	if err != nil {
		return err
	}
	if len(approved) == 0 {
		fmt.Println("Nothing to cleanup.")
		return nil
	}
	printer := tableprinter.New(os.Stdout)
	out := []pull{}
	for _, c := range approved {
		out = append(out, pull{
			URL:        c.Issue.GetHTMLURL(),
			LastUpdate: fmt.Sprintf("%s", humanize.Time(c.Issue.GetUpdatedAt())),
		})
	}
	printer.Print(out)

	fmt.Fprintf(os.Stderr, `
The cherry-pick-approved label will be removed from the pull requests listed above.

Do you wish to continue? (y/n): `)
	if !util.AskForConfirmation() {
		fmt.Println()
		os.Exit(0)
	}

	updater := github.NewPullRequestApprover(ctx, r.githubToken)
	for _, c := range approved {
		if err := updater.CherryPickRemove(ctx, c.Issue.GetHTMLURL()); err != nil {
			klog.Warningf("Failed to remove cherry-pick-approved from %s: %v", c.Issue.GetHTMLURL(), err)
			continue
		}
		updater.Comment(ctx, c.Issue.GetHTMLURL(), `:warning: The cherry-pick-approved label was removed by patch manager because this pull request failed to merge within approved merge window.

Next patch manager should investigate this and apply the label again, if the CI on this pull request is passing.'`)
		fmt.Fprintf(os.Stdout, "Removed cherry-pick-approved from %s and commented.\n", c.Issue.GetHTMLURL())
	}
	return nil
}

func stringifyLabels(labels []*githubapi.Label) string {
	out := []string{}
	for _, l := range labels {
		out = append(out, l.GetName())
	}
	return strings.Join(out, ",")
}
