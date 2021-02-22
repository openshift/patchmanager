package v1

type PatchManagerConfig struct {
	CapacityConfig     CapacityConfig   `yaml:"capacity"`
	ClassifiersConfigs ClassifierConfig `yaml:"classifiers"`
}

type ClassifierConfig struct {
	FlagsClassifier     FlagClassifierConfig      `yaml:"flags"`
	ComponentClassifier ComponentClassifierConfig `yaml:"components"`
	Severities          SeverityClassifierConfig  `yaml:"severities"`
	PMScores            PMScoreClassifierConfig   `yaml:"pmScores"`
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
