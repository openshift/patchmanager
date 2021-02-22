package api

import (
	"io/ioutil"

	"gopkg.in/yaml.v2"

	v1 "github.com/mfojtik/patchmanager/pkg/api/v1"
)

func ReadCapacityConfig(filename string) (*v1.CapacityConfig, error) {
	configBytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	var c v1.CapacityConfig
	if err := yaml.Unmarshal(configBytes, &c); err != nil {
		return nil, err
	}
	return &c, err
}

func ComponentCapacity(config *v1.CapacityConfig, componentName string) int {
	for _, c := range config.Components {
		if c.Name == componentName {
			return c.Capacity
		}
	}
	return config.DefaultCapacity
}
