package approve

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/openshift/patchmanager/pkg/config"

	"github.com/openshift/patchmanager/pkg/cmd/util"

	"github.com/openshift/patchmanager/pkg/github"

	v1 "github.com/openshift/patchmanager/pkg/api/v1"
	"gopkg.in/yaml.v2"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/klog/v2"
)

// approveOptions holds values to drive the start command.
type approveOptions struct {
	githubToken string
	force       bool
	inFile      string
	config      *config.PatchManagerConfig
	configFile  string
	comment     bool
}

// NewApproveCommand creates a render command.
func NewApproveCommand(ctx context.Context) *cobra.Command {
	runOpts := approveOptions{}
	cmd := &cobra.Command{
		Use:   "approve",
		Short: "Apply cherry-pick-approved label on pull request with 'pick' decision",
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
	fs.BoolVar(&r.force, "force", false, "Do not ask stupid questions and ship it")
	fs.BoolVar(&r.comment, "add-comment", false, "Provide informative comment about decision to all pull requests")
	fs.StringVar(&r.configFile, "config", os.Getenv("PATCHMANAGER_CONFIG"), "Path to a config file (PATCHMANAGER_CONFIG env variable)")
}

func (r *approveOptions) Validate() error {
	if len(r.githubToken) == 0 {
		return fmt.Errorf("github-token flag must be specified or GITHUB_TOKEN environment must be set")
	}
	if len(r.inFile) == 0 {
		return fmt.Errorf("candidate list file must be specified (-f)")
	}
	return nil
}

func (r *approveOptions) Complete() error {
	if len(r.githubToken) == 0 {
		r.githubToken = os.Getenv("GITHUB_TOKEN")
	}

	var err error
	if len(r.configFile) == 0 {
		return fmt.Errorf("you must provide valid config file (--config=config.yaml)")
	}
	r.config, err = config.GetConfig(r.configFile)
	if err != nil {
		return fmt.Errorf("unable to get config file %q: %v", r.configFile, err)
	}

	if !config.IsMergeWindowOpen(r.config.MergeWindowConfig) {
		fmt.Fprintf(os.Stderr, `# !!! WARNING !!!
#
# Based on the merge window configuration, approving pull requests is NOT recommended.
# The window opens on %s and close on %s.
# Please consult #forum-release for more details.

Do you wish to continue? (y/n): `, r.config.MergeWindowConfig.From, r.config.MergeWindowConfig.To)
		if !util.AskForConfirmation() {
			fmt.Println()
			os.Exit(0)
		}
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

	approver := github.NewPullRequestApprover(ctx, r.githubToken)

	if !r.force {
		fmt.Fprintf(os.Stdout, "Following Pull Requests will get cherry-pick-approved label:\n\n")
		for _, pr := range approved.Items {
			if pr.PullRequest.Decision != "pick" {
				continue
			}
			fmt.Fprintf(os.Stdout, "* %s\n", pr.PullRequest.URL)
		}

		fmt.Fprintf(os.Stdout, "\nDo you want to continue? (y/n)? ")
		if !util.AskForConfirmation() {
			return nil
		}
		fmt.Fprint(os.Stdout, "\n")
	}

	for _, pr := range approved.Items {
		if pr.PullRequest.Decision != "pick" {
			if r.comment {
				if err := approver.Comment(ctx, pr.PullRequest.URL, fmt.Sprintf(`
This pull request was not picked by patch manager for the current z-stream window (%s-%s).

Score: %.02f
Reason: %s

**NOTE**: This message is automatically generated, if you have questions please ask on #forum-release
`,
					pr.PullRequest.Score, pr.PullRequest.DecisionReason)); err != nil {
					klog.Errorf("Failed to comment on pull request %q: %v", pr.PullRequest.URL, err)
				}
			}
			continue
		}

		fmt.Fprintf(os.Stdout, "-> Approving %s ...\n", pr.PullRequest.URL)
		if err := approver.CherryPickApprove(ctx, pr.PullRequest.URL); err != nil {
			klog.Errorf("Failed to approve pull request %q: %v", pr.PullRequest.URL, err)
		}

		if r.comment {
			if err := approver.Comment(ctx, pr.PullRequest.URL, fmt.Sprintf("Approved for z-stream by score: %0.2f", pr.PullRequest.Score)); err != nil {
				klog.Errorf("Failed to comment on pull request %q: %v", pr.PullRequest.URL, err)
			}
		}
	}

	return nil
}
