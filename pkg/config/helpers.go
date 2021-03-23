package config

import (
	"crypto/tls"
	"io/ioutil"
	"net/http"
	"strings"

	"gopkg.in/yaml.v2"
)

// GetConfig gets a configuration file either locally or remotely via HTTP or HTTPS client.
func GetConfig(location string) (*PatchManagerConfig, error) {
	var config PatchManagerConfig

	// local files
	if !strings.HasPrefix(location, "http://") && !strings.HasPrefix(location, "https://") {
		configBytes, err := ioutil.ReadFile(location)
		if err != nil {
			return nil, err
		}
		err = yaml.Unmarshal(configBytes, &config)
		return &config, err
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	resp, err := client.Get(location)
	if err != nil {
		return nil, err
	}

	configBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(configBytes, &config)
	return &config, err
}

func ComponentCapacity(config *CapacityConfig, name string) (bool, int) {
	for _, group := range config.Groups {
		for _, c := range group.Components {
			if c == name {
				return true, group.Capacity
			}
		}
	}
	return false, config.MaximumDefaultPicksPerComponent
}
