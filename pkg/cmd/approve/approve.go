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
	skipComment string
	pickComment string
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
	fs.StringVar(&r.configFile, "config", os.Getenv("PATCHMANAGER_CONFIG"), "Path to a config file (PATCHMANAGER_CONFIG env variable)")
	fs.StringVar(&r.skipComment, "skip-comment", "", "Message to include in all skipped pull requests (if not set, no comment is made on skipping a PR)")
	fs.StringVar(&r.pickComment, "pick-comment", "", "Message to include in all picked pull requests (if not set, no comment is made on picking a PR)")
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

	if len(r.skipComment) == 0 {
		klog.Warning("WARNING: The --skip-comment flag is not used, no comment will be made to skipped PR's")
	}
	if len(r.pickComment) == 0 {
		klog.Warningf("WARNING: The --pick-comment flag is not used, no comment will be made to picked PR's")
	}

	return nil
}

func approvedPRs(prs v1.ApprovedCandidateList) []v1.ApprovedCandidate {
	result := []v1.ApprovedCandidate{}
	for i := range prs.Items {
		if prs.Items[i].PullRequest.Decision != "pick" {
			continue
		}
		result = append(result, prs.Items[i])
	}
	return result
}

func skippedPRs(prs v1.ApprovedCandidateList) []v1.ApprovedCandidate {
	result := []v1.ApprovedCandidate{}
	for i := range prs.Items {
		if prs.Items[i].PullRequest.Decision != "skip" {
			continue
		}
		result = append(result, prs.Items[i])
	}
	return result
}

func (r *approveOptions) Run(ctx context.Context) error {
	content, err := ioutil.ReadFile(r.inFile)
	if err != nil {
		return err
	}

	var prs v1.ApprovedCandidateList

	if err := yaml.Unmarshal(content, &prs); err != nil {
		return err
	}

	approved := approvedPRs(prs)
	skipped := skippedPRs(prs)
	if len(approved) == 0 && len(skipped) == 0 {
		return nil
	}

	approver := github.NewPullRequestApprover(ctx, r.githubToken)

	if !r.force {
		if len(approved) > 0 {
			fmt.Fprintf(os.Stdout, "Pull requests cherry-pick-prs:\n\n")
			for _, pr := range approved {
				fmt.Fprintf(os.Stdout, "* [%0.2f] %s\n", pr.PullRequest.Score, pr.PullRequest.URL)
			}
		}
		if len(skipped) > 0 {
			fmt.Fprintf(os.Stdout, "\nPull requests skipped:\n\n")
			for _, pr := range skipped {
				fmt.Fprintf(os.Stdout, "* [%0.2f] %s\n", pr.PullRequest.Score, pr.PullRequest.URL)
			}
		}
		fmt.Fprintf(os.Stdout, "\nDo you want to continue? (y/n)? ")
		if !util.AskForConfirmation() {
			return nil
		}
		fmt.Fprint(os.Stdout, "\n")
	}

	for _, pr := range skipped {
		// if there is no skip comment, skip commenting on PR
		if len(r.skipComment) == 0 {
			continue
		}
		mergeWindowMsg := ""
		if config.HasMergeWindow(r.config.MergeWindowConfig) {
			mergeWindowMsg = fmt.Sprintf(" (%s-%s)", r.config.MergeWindowConfig.From, r.config.MergeWindowConfig.To)
		}
		skipCommentMsg := "\n"
		if len(r.skipComment) > 0 {
			skipCommentMsg = fmt.Sprintf("\n*%s*\n", r.skipComment)
		}
		if err := approver.Comment(ctx, pr.PullRequest.URL, fmt.Sprintf(`
[patch-manager] :hourglass: This pull request was not picked by the patch manager for the current z-stream window and have to wait for the next window%s.
%s
* Score: *%0.2f*
* Reason: *%s*

**NOTE**: This message was automatically generated, if you have questions please ask on #forum-release
`,
			mergeWindowMsg, skipCommentMsg, pr.PullRequest.Score, pr.PullRequest.DecisionReason)); err != nil {
			klog.Errorf("Failed to comment on pull request %q: %v", pr.PullRequest.URL, err)
		}
		fmt.Fprintf(os.Stdout, "-> Commenting on skipping %s ...\n", pr.PullRequest.URL)
	}

	for _, pr := range approved {
		fmt.Fprintf(os.Stdout, "-> Approving %s ...\n", pr.PullRequest.URL)
		if err := approver.CherryPickApprove(ctx, pr.PullRequest.URL); err != nil {
			klog.Errorf("Failed to approve pull request %q: %v", pr.PullRequest.URL, err)
		}

		// if there is no pick comment, skip commenting on PR
		if len(r.pickComment) == 0 {
			continue
		}
		pickCommentMsg := ""
		if len(r.pickComment) > 0 {
			pickCommentMsg = fmt.Sprintf("\n\n%s\n", r.pickComment)
		}

		fmt.Fprintf(os.Stdout, "-> Commenting on approval of %s ...\n", pr.PullRequest.URL)
		if err := approver.Comment(ctx, pr.PullRequest.URL, fmt.Sprintf("[patch-manager] :rocket: Approved for z-stream by score: %0.2f%s", pr.PullRequest.Score, pickCommentMsg)); err != nil {
			klog.Errorf("Failed to comment on pull request %q: %v", pr.PullRequest.URL, err)
		}
	}
	fmt.Println()

	return nil
}
