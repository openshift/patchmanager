package run

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/cheggaaa/pb/v3"
	"github.com/lensesio/tableprinter"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"gopkg.in/yaml.v2"
	"k8s.io/klog/v2"

	"github.com/openshift/patchmanager/pkg/api"
	v1 "github.com/openshift/patchmanager/pkg/api/v1"
	"github.com/openshift/patchmanager/pkg/classifiers"
	"github.com/openshift/patchmanager/pkg/config"
	"github.com/openshift/patchmanager/pkg/github"
	"github.com/openshift/patchmanager/pkg/scoring"
)

// runOptions holds values to drive the start command.
type runOptions struct {
	bugzillaAPIKey string
	githubToken    string
	release        string
	outFile        string

	maxPicks   int
	configFile string
	config     *config.PatchManagerConfig

	classifier classifiers.Classifier
}

// NewRunCommand creates a render command.
func NewRunCommand(ctx context.Context) *cobra.Command {
	runOpts := runOptions{}
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run z-stream pull requests classification for given release",
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
	fs.StringVar(&r.configFile, "config", os.Getenv("PATCHMANAGER_CONFIG"), "Path to a config file (PATCHMANAGER_CONFIG env variable)")
	fs.IntVar(&r.maxPicks, "max-pick", 10, "Set the default maxPicks to approve if config file is not used or default maxPicks is not set")
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
		return fmt.Errorf("release flag must be set (eg: --release=4.7)")
	}
	if r.maxPicks <= 0 {
		return fmt.Errorf("maxPicks must be above 0")
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

	var err error
	if len(r.configFile) == 0 {
		return fmt.Errorf("you must provide valid config file (--config=config.yaml)")
	}
	r.config, err = config.GetConfig(r.configFile)
	if err != nil {
		return fmt.Errorf("unable to get config file %q: %v", r.configFile, err)
	}

	r.classifier = classifiers.NewMultiClassifier(
		&classifiers.SeverityClassifier{Config: &r.config.ClassifiersConfigs.Severities},
		&classifiers.ComponentClassifier{Config: &r.config.ClassifiersConfigs.ComponentClassifier},
		&classifiers.KeywordsClassifier{Config: &r.config.ClassifiersConfigs.KeywordsClassifier},
		&classifiers.ProductManagementScoreClassifier{Config: &r.config.ClassifiersConfigs.PMScores},
	)
	return nil
}

type capacityTracker struct {
	config           *config.CapacityConfig
	componentPicks   map[string]int
	componentSkips   map[string]int
	componentCounter map[string]int
}

func (c capacityTracker) inc(component string) {
	c.componentCounter[component] = c.componentCounter[component] + 1
}

func (c capacityTracker) hasCapacity(component string) bool {
	isConfigured, maxComponentCapacity := config.ComponentCapacity(c.config, component)
	if isConfigured {
		return c.componentCounter[component] <= maxComponentCapacity
	}
	return c.componentCounter[component] <= c.config.MaximumDefaultPicksPerComponent
}

func componentName(p []string) string {
	return strings.ToLower(strings.Join(p, "/"))
}

func (r *runOptions) Run(ctx context.Context) error {
	pullsToReview, err := github.NewPullRequestLister(ctx, r.githubToken, r.bugzillaAPIKey).ListForRelease(ctx, r.release)
	if err != nil {
		return err
	}

	if capacity := len(r.config.CapacityConfig.Groups); capacity > 0 {
		klog.Infof("Capacity configuration for %d groups loaded (default per component: %d)", capacity, r.config.CapacityConfig.MaximumDefaultPicksPerComponent)
	}
	klog.Infof("Maximum allowed pull requests to pick is %d", r.maxPicks)

	// assign score to each pull request by running it trough set of classifiers
	progress := pb.StartNew(len(pullsToReview))
	pool := scoring.NewWorkerPool(r.classifier).WithCallback(func(interface{}) {
		progress.Increment()
	})
	if err := pool.Add(pullsToReview...); err != nil {
		return err
	}

	klog.Infof("Wait to finish classifying %d z-stream candidate pull requests ...", len(pullsToReview))
	if err := pool.WaitForFinish(); err != nil {
		return err
	}
	progress.Finish()

	// order the pending pull requests by score
	sort.Slice(pullsToReview, func(i, j int) bool {
		return pullsToReview[i].Score > pullsToReview[j].Score
	})

	capacity := &capacityTracker{
		config:           &r.config.CapacityConfig,
		componentCounter: map[string]int{},
		componentPicks:   map[string]int{},
		componentSkips:   map[string]int{},
	}

	// decide which pull requests we are going to pick based on the componentCounter
	candidates := []v1.Candidate{}
	totalPicks := 0

	for _, p := range pullsToReview {
		decision := "pick"
		decisionReason := fmt.Sprintf("picked for z-stream with score %0.2f", p.Score)

		// increment capacity counter for this component
		capacity.inc(componentName(p.Bug().Component))

		if p.Score < 0.0 {
			// if the component has a negative score, it is unlikely this PR is meeting a important criteria
			decision = "skip"
			decisionReason = fmt.Sprintf("automated classifiers have given this PR a negative score meaning that " +
				"it does not meet important merge criteria for this release; if you believe this PR is an exception, " +
				"please contact @patch-manager in coreos Slack")
		} else if !capacity.hasCapacity(componentName(p.Bug().Component)) {
			// if component has no capacity to take this pick
			decision = "skip"
			_, componentCapacity := config.ComponentCapacity(&r.config.CapacityConfig, componentName(p.Bug().Component))
			decisionReason = fmt.Sprintf("maximum allowed picks for component %s is %d", componentName(p.Bug().Component), componentCapacity)
		}

		// if there are more picks than total picks allowed
		if decision == "pick" {
			totalPicks++
		}
		if totalPicks > r.maxPicks {
			decision = "skip"
			decisionReason = fmt.Sprintf("maximum total for this z-stream is %d", r.maxPicks)
		}

		if decision == "pick" {
			capacity.componentPicks[componentName(p.Bug().Component)]++
		} else {
			capacity.componentSkips[componentName(p.Bug().Component)]++
		}

		// add to candidate list
		candidates = append(candidates, v1.Candidate{
			PMScore:        p.Bug().PMScore,
			Score:          p.Score,
			Description:    p.Bug().Summary,
			PullRequestURL: p.Issue.GetHTMLURL(),
			BugNumber:      fmt.Sprintf("%d", p.Bug().ID),
			Component:      componentName(p.Bug().Component),
			Severity:       p.Bug().Severity,
			Decision:       decision,
			DecisionReason: decisionReason,
		})
	}

	out, err := yaml.Marshal(api.NewCandidateList(candidates))
	if err != nil {
		return err
	}

	metrics := []componentMetric{}
	for name, count := range capacity.componentCounter {
		metrics = append(metrics, componentMetric{
			Component: name,
			Total:     count,
			Picks:     capacity.componentPicks[name],
			Skips:     capacity.componentSkips[name],
		})
	}
	printer := tableprinter.New(os.Stdout)
	fmt.Println()
	printer.Print(metrics)
	fmt.Println()

	output := os.Stdout
	if len(r.outFile) > 0 {
		output, err = os.OpenFile(r.outFile, os.O_CREATE|os.O_WRONLY, 0666)
		if err != nil {
			return err
		}
		klog.Infof("Result saved to %q", r.outFile)
	}

	if _, err = fmt.Fprintf(output, "%s\n", string(out)); err != nil {
		return err
	}
	if len(r.outFile) > 0 {
		return output.Sync()
	}

	return nil
}

type componentMetric struct {
	Component string `header:"Component Name"`
	Total     int    `header:"Total"`
	Picks     int    `header:"Picks"`
	Skips     int    `header:"Skips"`
}
