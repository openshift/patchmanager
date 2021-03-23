package config

import (
	"crypto/tls"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"k8s.io/klog/v2"

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

func IsMergeWindowOpen(c MergeWindowConfig) bool {
	// no configuration means always open
	if len(c.From) == 0 || len(c.To) == 0 {
		return true
	}

	from, err := time.Parse("2006-01-02", c.From)
	if err != nil {
		klog.Warning("Invalid merge window from time configuration: %q, expected format: 2006-01-02 (%s)", c.From, err)
	}
	to, err := time.Parse("2006-01-02", c.To)
	if err != nil {
		klog.Warning("Invalid merge window to time configuration: %q, expected format: 2006-01-02 (%s)", c.To, err)
	}

	return time.Now().Before(to) && time.Now().After(from)
}
