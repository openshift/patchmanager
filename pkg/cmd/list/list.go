package list

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/fatih/color"

	v1 "github.com/openshift/patchmanager/pkg/api/v1"
	"gopkg.in/yaml.v2"

	"github.com/lensesio/tableprinter"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/klog/v2"
)

// listOptions holds values to drive the start command.
type listOptions struct {
	inputFile string
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
	fs.StringVarP(&r.inputFile, "file", "f", "", "Set input file to read the list of candidates")
}

func (r *listOptions) Validate() error {
	if len(r.inputFile) == 0 {
		return fmt.Errorf("input file must be specified")
	}
	return nil
}

func (r *listOptions) Complete() error {
	return nil
}

type pull struct {
	URL      string  `header:"URL"`
	Score    float32 `header:"Score"`
	Decision string  `header:"Decision"`
	Reason   string  `header:"Reason"`
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
	content, err := ioutil.ReadFile(r.inputFile)
	if err != nil {
		return err
	}
	var candidates v1.ApprovedCandidateList
	if err := yaml.Unmarshal(content, &candidates); err != nil {
		return err
	}
	printer := tableprinter.New(os.Stdout)

	out := []pull{}
	for _, c := range candidates.Items {
		out = append(out, pull{
			URL:      c.PullRequest.URL,
			Decision: colorizeDecision(c.PullRequest.Decision),
			Reason:   c.PullRequest.DecisionReason,
			Score:    c.PullRequest.Score,
		})
	}
	printer.Print(out)

	return nil
}
