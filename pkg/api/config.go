package api

import (
	"crypto/tls"
	"io/ioutil"
	"net/http"
	"strings"

	v1 "github.com/openshift/patchmanager/pkg/api/v1"
	"gopkg.in/yaml.v2"
)

// GetConfig gets a configuration file either locally or remotely via HTTP or HTTPS client.
func GetConfig(location string) (*v1.PatchManagerConfig, error) {
	var config v1.PatchManagerConfig

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
