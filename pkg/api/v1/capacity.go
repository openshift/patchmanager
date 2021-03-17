package v1

// CapacityConfig define a list of component and their QE capacity
type CapacityConfig struct {
	Groups []ComponentGroup `yaml:"groups"`

	// DefaultCapacity is capacity used when there is no capacity defined for given component
	DefaultCapacity int `yaml:"default"`
}

type ComponentGroup struct {
	Name       string   `yaml:"name"`
	Capacity   int      `yaml:"capacity"`
	Components []string `yaml:"components"`
}
