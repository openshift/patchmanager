package config

type PatchManagerConfig struct {
	Release            string            `yaml:"release"`
	CapacityConfig     CapacityConfig    `yaml:"capacity"`
	ClassifiersConfigs ClassifierConfig  `yaml:"classifiers"`
	MergeWindowConfig  MergeWindowConfig `yaml:"mergeWindow"`
}

type ClassifierConfig struct {
	FlagsClassifier     FlagClassifierConfig      `yaml:"flags"`
	ComponentClassifier ComponentClassifierConfig `yaml:"components"`
	Severities          SeverityClassifierConfig  `yaml:"severities"`
	PMScores            PMScoreClassifierConfig   `yaml:"pmScores"`
}

type MergeWindowConfig struct {
	From string `yaml:"from,omitempty"`
	To   string `yaml:"to,omitempty"`
}

type FlagClassifierConfig map[string]float32
type ComponentClassifierConfig map[string]float32
type SeverityClassifierConfig map[string]float32

type PMScoreClassifierConfig []PMScoreRange

type PMScoreRange struct {
	From  int     `yaml:"from"`
	To    int     `yaml:"to"`
	Score float32 `yaml:"score"`
}

// CapacityConfig define a list of component and their QE capacity
type CapacityConfig struct {
	Groups []ComponentGroup `yaml:"groups"`

	// MaximumTotalPicks is total number if pull request approved regardless of component
	MaximumTotalPicks int `yaml:"maxTotalPicks"`

	// MaximumdefaultPicksPerComponent is default capacity for component when there is no capacity defined.
	MaximumDefaultPicksPerComponent int `yaml:"maxDefaultPicksPerComponent"`
}

type ComponentGroup struct {
	Name       string   `yaml:"name"`
	Capacity   int      `yaml:"capacity"`
	Components []string `yaml:"components"`
}
