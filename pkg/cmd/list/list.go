package list

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/openshift/patchmanager/pkg/config"

	githubapi "github.com/google/go-github/v32/github"

	"github.com/dustin/go-humanize"
	"github.com/fatih/color"
	"github.com/lensesio/tableprinter"
	v1 "github.com/openshift/patchmanager/pkg/api/v1"
	"github.com/openshift/patchmanager/pkg/github"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"gopkg.in/yaml.v2"
	"k8s.io/klog/v2"
)

// listOptions holds values to drive the start command.
type listOptions struct {
	inputFile      string
	candidates     bool
	approved       bool
	release        string
	githubToken    string
	bugzillaAPIKey string
	config         *config.PatchManagerConfig
	configFile     string
}

// NewListCommand creates a render command.
func NewListCommand(ctx context.Context) *cobra.Command {
	runOpts := listOptions{}
	cmd := &cobra.Command{
		Use:   "list",
		Short: "Print table from pull request YAML file",
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

func (r *listOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&r.githubToken, "github-token", "", "Github Access Token (GITHUB_TOKEN env variable)")
	fs.StringVar(&r.bugzillaAPIKey, "bugzilla-apikey", "", "Bugzilla API Key (BUGZILLA_APIKEY env variable)")
	fs.StringVarP(&r.inputFile, "file", "f", "", "Set input file to read the list of candidates")
	fs.StringVar(&r.release, "release", "", "Release to use to list candidates")
	fs.BoolVar(&r.candidates, "candidates", false, "List candidate PR's for a release")
	fs.BoolVar(&r.approved, "approved", false, "List approved PR's for a release")
	fs.StringVar(&r.configFile, "config", os.Getenv("PATCHMANAGER_CONFIG"), "Path to a config file (PATCHMANAGER_CONFIG env variable)")
}

func (r *listOptions) Validate() error {
	if !r.approved && !r.candidates && len(r.inputFile) == 0 {
		return fmt.Errorf("input file must be specified")
	}

	return nil
}

func (r *listOptions) Complete() error {
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
	if r.approved || r.candidates {
		if len(r.release) == 0 {
			return fmt.Errorf("you must specify target release to list pr's (eg. --release=4.6)")
		}
	}
	return nil
}

type approvedPull struct {
	URL      string  `header:"URL"`
	Score    float32 `header:"Score"`
	Decision string  `header:"Decision"`
	Reason   string  `header:"Reason"`
}

type pull struct {
	URL    string `header:"URL"`
	Status string `header:"Status"`
}

func colorizeDecision(d string) string {
	switch d {
	case "skip":
		return color.RedString("skip")
	case "pick":
		return color.GreenString("pick")
	default:
		return d
	}
}

func (r *listOptions) Run(ctx context.Context) error {
	if r.approved {
		return r.RunListApproved(ctx)
	}
	if r.candidates {
		return r.RunListCandidates(ctx)
	}
	content, err := ioutil.ReadFile(r.inputFile)
	if err != nil {
		return err
	}
	var candidates v1.ApprovedCandidateList
	if err := yaml.Unmarshal(content, &candidates); err != nil {
		return err
	}
	printer := tableprinter.New(os.Stdout)

	out := []approvedPull{}
	for _, c := range candidates.Items {
		out = append(out, approvedPull{
			URL:      c.PullRequest.URL,
			Decision: colorizeDecision(c.PullRequest.Decision),
			Reason:   c.PullRequest.DecisionReason,
			Score:    c.PullRequest.Score,
		})
	}
	printer.Print(out)

	return nil
}

func stringifyLabels(labels []*githubapi.Label) string {
	out := []string{}
	for _, l := range labels {
		out = append(out, l.GetName())
	}
	return strings.Join(out, ",")
}

func (r *listOptions) RunListApproved(ctx context.Context) error {
	approved, err := github.NewPullRequestLister(ctx, r.githubToken, r.bugzillaAPIKey).ListApprovedForRelease(ctx, r.release)
	if err != nil {
		return err
	}
	printer := tableprinter.New(os.Stdout)
	out := []pull{}
	for _, c := range approved {
		out = append(out, pull{
			URL:    c.Issue.GetHTMLURL(),
			Status: fmt.Sprintf("lastUpdate=%v\nlabels=%s", humanize.Time(c.Issue.GetUpdatedAt()), stringifyLabels(c.Issue.Labels)),
		})
	}
	printer.Print(out)
	return nil
}

func (r *listOptions) RunListCandidates(ctx context.Context) error {
	candidates, err := github.NewPullRequestLister(ctx, r.githubToken, r.bugzillaAPIKey).ListCandidatesForRelease(ctx, r.release)
	if err != nil {
		return err
	}
	printer := tableprinter.New(os.Stdout)
	out := []pull{}
	for _, c := range candidates {
		out = append(out, pull{
			URL:    c.Issue.GetHTMLURL(),
			Status: fmt.Sprintf("lastUpdate=%v", humanize.Time(c.Issue.GetUpdatedAt())),
		})
	}
	printer.Print(out)
	return nil
}
