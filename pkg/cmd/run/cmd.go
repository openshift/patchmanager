package run

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"strings"

	"github.com/mfojtik/patchmanager/pkg/api"
	"gopkg.in/yaml.v2"

	v1 "github.com/mfojtik/patchmanager/pkg/api/v1"

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
	outFile        string

	capacity   int
	configFile string
	config     *v1.PatchManagerConfig

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
	fs.StringVar(&r.configFile, "config", "", "Path to a config file")
	fs.IntVar(&r.capacity, "capacity", 10, "Set the default capacity to approve if config file is not used or default capacity is not set")
	fs.StringVarP(&r.outFile, "output", "o", "", "Set output file instead of standard output")
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
	if r.capacity <= 0 {
		return fmt.Errorf("capacity must be above 0")
	}
	if len(r.configFile) == 0 {
		return fmt.Errorf("need to specify valid config file")
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

	configBytes, err := ioutil.ReadFile(r.configFile)
	if err != nil {
		return err
	}

	r.config = &v1.PatchManagerConfig{}
	if err := yaml.Unmarshal(configBytes, r.config); err != nil {
		return err
	}

	r.classifier = classifier.New(
		&classifier.SeverityClassifier{Config: &r.config.ClassifiersConfigs.Severities},
		&classifier.ComponentClassifier{Config: &r.config.ClassifiersConfigs.ComponentClassifier},
		&classifier.FlagsClassifier{Config: &r.config.ClassifiersConfigs.FlagsClassifier},
		&classifier.ProductManagementScoreClassifier{Config: &r.config.ClassifiersConfigs.PMScores},
	)
	return nil
}

type capacityTracker struct {
	config   *v1.CapacityConfig
	capacity map[string]int
}

func (c capacityTracker) hasCapacity(component string) bool {
	current, ok := c.capacity[component]
	if !ok {
		c.capacity[component] = c.config.DefaultCapacity - 1
		return true
	}
	if current <= api.ComponentCapacity(c.config, component) {
		c.capacity[component] = c.capacity[component] - 1
		return true
	}
	return false
}

func (r *runOptions) Run(ctx context.Context) error {
	pendingPullRequests, err := github.NewPullRequestLister(ctx, r.githubToken, r.bugzillaAPIKey).ListForRelease(ctx, r.release)
	if err != nil {
		return err
	}

	if capacity := len(r.config.CapacityConfig.Components); capacity > 0 {
		klog.Infof("Capacity configuration for %d components loaded (default capacity: %d)", capacity, r.config.CapacityConfig.DefaultCapacity)
	} else {
		klog.Infof("Using default capacity %d", r.capacity)
	}

	klog.Infof("Wait to finish processing %d cherry-pick-approve pull requests ...", len(pendingPullRequests))

	for i := range pendingPullRequests {
		pendingPullRequests[i].Score = r.classifier.Score(pendingPullRequests[i])
	}

	sort.Slice(pendingPullRequests, func(i, j int) bool {
		return pendingPullRequests[i].Score > pendingPullRequests[j].Score
	})

	capacity := &capacityTracker{
		config:   &r.config.CapacityConfig,
		capacity: map[string]int{},
	}

	candidates := []v1.Candidate{}
	for _, p := range pendingPullRequests {
		decision := "pick"
		decisionReason := ""
		if !capacity.hasCapacity(strings.Join(p.Bug().Component, "/")) {
			decision = "skip"
			decisionReason = fmt.Sprintf("target capacity for component %s is %d", strings.Join(p.Bug().Component, "/"), api.ComponentCapacity(&r.config.CapacityConfig, strings.Join(p.Bug().Component, "/")))
		}
		candidates = append(candidates, v1.Candidate{
			PMScore:        p.Bug().PMScore,
			Score:          p.Score,
			Description:    p.Bug().Summary,
			PullRequestURL: p.Issue.GetHTMLURL(),
			BugNumber:      fmt.Sprintf("%d", p.Bug().ID),
			Component:      strings.Join(p.Bug().Component, "/"),
			Severity:       p.Bug().Severity,
			Decision:       decision,
			DecisionReason: decisionReason,
		})
	}

	out, err := yaml.Marshal(api.NewCandidateList(candidates))
	if err != nil {
		return err
	}

	output := os.Stdout
	if len(r.outFile) > 0 {
		output, err = os.OpenFile(r.outFile, os.O_CREATE|os.O_WRONLY, 0666)
		if err != nil {
			return err
		}
	}

	if _, err = fmt.Fprintf(output, "%s\n", string(out)); err != nil {
		return err
	}

	return output.Sync()
}
