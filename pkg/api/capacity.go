package api

import (
	v1 "github.com/mfojtik/patchmanager/pkg/api/v1"
)

func ComponentCapacity(config *v1.CapacityConfig, name string) (bool, int) {
	for _, group := range config.Groups {
		for _, c := range group.Components {
			if c == name {
				return true, group.Capacity
			}
		}
	}
	return false, config.MaximumDefaultPicksPerComponent
}
