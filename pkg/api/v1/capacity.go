package v1

// CapacityConfig define a list of component and their QE capacity
type CapacityConfig struct {
	Components []ComponentCapacity `yaml:"components"`

	// DefaultCapacity is capacity used when there is no capacity defined for given component
	DefaultCapacity int `yaml:"default"`
}

type ComponentCapacity struct {
	Name     string `yaml:"name"`
	Capacity int    `yaml:"capacity"`
}
