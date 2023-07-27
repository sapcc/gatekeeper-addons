/*******************************************************************************
*
* Copyright 2023 SAP SE
*
* Licensed under the Apache License, Version 2.0 (the "License");
* you may not use this file except in compliance with the License.
* You should have received a copy of the License along with this
* program. If not, you may obtain a copy of the License at
*
*     http://www.apache.org/licenses/LICENSE-2.0
*
* Unless required by applicable law or agreed to in writing, software
* distributed under the License is distributed on an "AS IS" BASIS,
* WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
* See the License for the specific language governing permissions and
* limitations under the License.
*
*******************************************************************************/

package main

import (
	"errors"
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
)

// Configuration contains the contents of the config file.
type Configuration struct {
	ClusterIdentity map[string]string `yaml:"cluster_identity"`
	Kubernetes      struct {
		KubeconfigPath string `yaml:"kubeconfig"`
		Context        string `yaml:"context"`
	} `yaml:"kubernetes"`
	Metrics struct {
		ListenAddress string `yaml:"listen_address"`
	} `yaml:"metrics"`
	MergingRules    any `yaml:"merging_rules"`    //TODO type
	ProcessingRules any `yaml:"processing_rules"` //TODO type
	Swift           struct {
		ContainerName string `yaml:"container_name"`
		ObjectName    string `yaml:"object_name"`
	} `yaml:"swift"`
}

// ReadConfiguration reads the config file at the given path.
func ReadConfiguration(configPath string) (Configuration, error) {
	buf, err := os.ReadFile(configPath)
	if err != nil {
		return Configuration{}, err
	}

	var cfg Configuration
	err = yaml.UnmarshalStrict(buf, &cfg)
	if err != nil {
		return Configuration{}, fmt.Errorf("while parsing %s: %w", configPath, err)
	}

	//apply default values, check for universally required values
	if cfg.Metrics.ListenAddress == "" {
		cfg.Metrics.ListenAddress = ":8080"
	}
	if len(cfg.ClusterIdentity) == 0 {
		return Configuration{}, errors.New("missing required configuration value: cluster_identity")
	}

	return cfg, nil
}
