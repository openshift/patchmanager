package v1

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
