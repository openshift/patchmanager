package api

import (
	v1 "github.com/mfojtik/patchmanager/pkg/api/v1"
)

func ComponentCapacity(config *v1.CapacityConfig, name string) int {
	for _, group := range config.Groups {
		for _, c := range group.Components {
			if c == name {
				return group.Capacity
			}
		}
	}
	return config.DefaultCapacity
}
