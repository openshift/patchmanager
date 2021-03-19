package approve

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

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
		if !askForConfirmation() {
			return nil
		}
		fmt.Fprint(os.Stdout, "\n")
	}

	for _, pr := range approved.Items {
		if pr.PullRequest.Decision != "pick" {
			continue
		}
		fmt.Fprintf(os.Stdout, "- Approving %s ...\n", pr.PullRequest.URL)
		if err := approver.CherryPickApprove(ctx, pr.PullRequest.URL); err != nil {
			klog.Errorf("Failed to approve pull request %q: %v", err)
		}
	}

	return nil
}

func askForConfirmation() bool {
	var response string
	_, err := fmt.Scanln(&response)
	if err != nil {
		klog.Fatal(err)
	}

	switch strings.ToLower(response) {
	case "y", "yes":
		return true
	case "n", "no":
		return false
	default:
		fmt.Println("I'm sorry but I didn't get what you meant, please type (y)es or (n)o and then press enter:")
		return askForConfirmation()
	}
}
