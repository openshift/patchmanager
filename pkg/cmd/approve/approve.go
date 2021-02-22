package approve

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"

	v1 "github.com/mfojtik/patchmanager/pkg/api/v1"
	"gopkg.in/yaml.v2"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/klog/v2"
)

// approveOptions holds values to drive the start command.
type approveOptions struct {
	githubToken string
	inFile      string
}

// NewApproveCommand creates a render command.
func NewApproveCommand(ctx context.Context) *cobra.Command {
	runOpts := approveOptions{}
	cmd := &cobra.Command{
		Use: "approve",
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

func (r *approveOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&r.githubToken, "github-token", "", "Github Access Token (GITHUB_TOKEN env variable)")
	fs.StringVarP(&r.inFile, "file", "f", "", "Set input file to read the list of candidates")
}

func (r *approveOptions) Validate() error {
	if len(r.githubToken) == 0 {
		return fmt.Errorf("github-token flag must be specified or GITHUB_TOKEN environment must be set")
	}
	if len(r.inFile) == 0 {
		return fmt.Errorf("input file must be specified")
	}
	return nil
}

func (r *approveOptions) Complete() error {
	if len(r.githubToken) == 0 {
		r.githubToken = os.Getenv("GITHUB_TOKEN")
	}
	return nil
}

func (r *approveOptions) Run(ctx context.Context) error {
	content, err := ioutil.ReadFile(r.inFile)
	if err != nil {
		return err
	}

	var approved v1.ApprovedCandidateList

	if err := yaml.Unmarshal(content, &approved); err != nil {
		return err
	}
	// approver := github.NewPullRequestApprover(ctx, r.githubToken)

	for _, pr := range approved.Items {
		if pr.PullRequest.Decision != "pick" {
			continue
		}
		fmt.Fprintf(os.Stdout, "Approving pull request %s ...\n", pr.PullRequest.URL)
		/*
			if err := approver.CherryPickApprove(ctx, pr.PullRequest.URL); err != nil {
				log.Errorf("Failed to approve pull request %q: %v", err)
			}
		*/
	}

	return nil
}
