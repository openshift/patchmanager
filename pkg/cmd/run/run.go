package run

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"strings"

	"github.com/openshift/patchmanager/pkg/rule"

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

	configFile string
	config     *config.PatchManagerConfig

	useCapacityPercent int
	useCapacityCount   int

	classifier classifiers.Classifier
	rules      rule.Ruler
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
	fs.StringVarP(&r.outFile, "output", "o", "", "Set output file instead of standard output")
	fs.IntVar(&r.useCapacityPercent, "use-capacity-percent", 100, "How much capacity should be used to pick PR's (0-100)")
}

func (r *runOptions) Validate() error {
	if len(r.bugzillaAPIKey) == 0 {
		return fmt.Errorf("bugzilla-apikey flag must be specified or BUGZILLA_APIKEY environment must be set")
	}
	if len(r.githubToken) == 0 {
		return fmt.Errorf("github-token flag must be specified or GITHUB_TOKEN environment must be set")
	}
	if len(r.configFile) == 0 {
		return fmt.Errorf("need to specify valid config file")
	}
	if r.useCapacityPercent > 100 || r.useCapacityPercent < 0 {
		return fmt.Errorf("use-capacity-percent value must be between 0 and 100 (percent)")
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
	if len(r.config.Release) > 0 && len(r.release) == 0 {
		r.release = r.config.Release
	}

	r.classifier = classifiers.NewMultiClassifier(
		&classifiers.SeverityClassifier{Config: &r.config.ClassifiersConfigs.Severities},
		&classifiers.ComponentClassifier{Config: &r.config.ClassifiersConfigs.ComponentClassifier},
		&classifiers.KeywordsClassifier{Config: &r.config.ClassifiersConfigs.KeywordsClassifier},
		&classifiers.ProductManagementScoreClassifier{Config: &r.config.ClassifiersConfigs.PMScores},
	)

	r.rules = rule.NewMultiRuler(
		&rule.PullRequestLabelRule{Config: &r.config.RulesConfig.PullRequestLabelConfig},
	)

	// calculate how much PR's would be approved based on the "use capacity" percent
	r.useCapacityCount = int((float32(r.config.CapacityConfig.MaximumTotalPicks) * 0.01) * float32(r.useCapacityPercent))
	klog.Infof("Using %d%% of total QE capacity of %d PR's approved for ALL z-stream releases (max. %d picked)", r.useCapacityPercent, r.config.CapacityConfig.MaximumTotalPicks, r.useCapacityCount)

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
	pullsToReview, err := github.NewPullRequestLister(ctx, r.githubToken, r.bugzillaAPIKey).ListCandidatesForRelease(ctx, r.release)
	if err != nil {
		return err
	}

	if capacity := len(r.config.CapacityConfig.Groups); capacity > 0 {
		klog.Infof("Capacity configuration for %d groups loaded (default per component: %d)", capacity, r.config.CapacityConfig.MaximumDefaultPicksPerComponent)
	}

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

	candidates := []v1.Candidate{}
	capacity := &capacityTracker{
		config:           &r.config.CapacityConfig,
		componentCounter: map[string]int{},
		componentPicks:   map[string]int{},
		componentSkips:   map[string]int{},
	}

	pullsToClassify := []*github.PullRequest{}
	for i, p := range pullsToReview {
		capacity.componentCounter[componentName(p.Bug().Component)] = 0
		decisions, ok := r.rules.Evaluate(pullsToReview[i])
		if ok {
			pullsToClassify = append(pullsToClassify, pullsToReview[i])
			continue
		}
		candidates = append(candidates, v1.Candidate{
			PMScore:        p.Bug().PMScore,
			Score:          0,
			Description:    p.Bug().Summary,
			PullRequestURL: p.Issue.GetHTMLURL(),
			BugNumber:      fmt.Sprintf("%d", p.Bug().ID),
			Component:      componentName(p.Bug().Component),
			Severity:       p.Bug().Severity,
			Decision:       "skip",
			DecisionReason: strings.Join(decisions, ", "),
		})
		capacity.componentSkips[componentName(p.Bug().Component)]++
	}
	klog.Infof("%d pull requests refused by the rules", len(candidates))

	// order the pending pull requests by score
	sort.Slice(pullsToClassify, func(i, j int) bool {
		return pullsToClassify[i].Score > pullsToClassify[j].Score
	})

	// decide which pull requests we are going to pick based on the componentCounter
	totalPicks := 0

	for _, p := range pullsToClassify {
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
		if totalPicks > r.useCapacityCount {
			decision = "skip"
			decisionReason = fmt.Sprintf("maximum QE capacity for all z-stream is %d", r.config.CapacityConfig.MaximumTotalPicks)
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
	for name, _ := range capacity.componentCounter {
		metrics = append(metrics, componentMetric{
			Component: name,
			Total:     capacity.componentPicks[name] + capacity.componentSkips[name],
			Picks:     capacity.componentPicks[name],
			Skips:     capacity.componentSkips[name],
		})
	}
	printer := tableprinter.New(os.Stdout)
	fmt.Println()
	printer.Print(metrics)
	fmt.Println()

	// to file
	if len(r.outFile) > 0 {
		if err := ioutil.WriteFile(r.outFile, out, os.ModePerm); err != nil {
			return err
		}
		klog.Infof("Result saved to %q", r.outFile)
		return nil
	}

	// standard output
	if _, err = fmt.Fprintf(os.Stdout, "%s\n", string(out)); err != nil {
		return err
	}
	return nil
}

type componentMetric struct {
	Component string `header:"Component Name"`
	Total     int    `header:"Total"`
	Picks     int    `header:"Picks"`
	Skips     int    `header:"Skips"`
}
