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

	"github.com/sapcc/go-bits/errext"
	"github.com/sapcc/go-bits/regexpext"
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
	MergingRules    []Rule             `yaml:"merging_rules"`
	ProcessingRules []Rule             `yaml:"processing_rules"`
	Swift           SwiftConfiguration `yaml:"swift"`
}

// Rule is a rule that can appear in `processing_rules` or `merging_rules`.
type Rule struct {
	Description string                             `yaml:"description"`
	Match       map[string]regexpext.BoundedRegexp `yaml:"match"`
	Replace     ReplaceRule                        `yaml:"replace"`
}

// ReplaceRule appears in type Rule.
type ReplaceRule struct {
	Source  string                  `yaml:"source"`
	Pattern regexpext.BoundedRegexp `yaml:"pattern"`
	Target  map[string]string       `yaml:"target"`
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

	// apply default values, check for universally required values
	if cfg.Metrics.ListenAddress == "" {
		cfg.Metrics.ListenAddress = ":8080"
	}
	if len(cfg.ClusterIdentity) == 0 {
		return Configuration{}, errors.New("missing required configuration value: cluster_identity")
	}

	return cfg, nil
}

// ValidateRules returns a list of validation errors for the configuration's
// MergingRules and ProcessingRules.
func (cfg Configuration) ValidateRules() (errs errext.ErrorSet) {
	for idx, rule := range cfg.ProcessingRules {
		errs.Append(rule.validate(fmt.Sprintf("processing_rules[%d]", idx)))
	}
	for idx, rule := range cfg.MergingRules {
		errs.Append(rule.validate(fmt.Sprintf("merging_rules[%d]", idx)))
	}
	return
}

func (r Rule) validate(path string) (errs errext.ErrorSet) {
	for key, pattern := range r.Match {
		if pattern == "" {
			errs.Addf("empty regex in %s.match[%q] (rule %q) will probably not do what you think (if you actually want to match empty strings only, write `^$` to confirm your intention)", key, path, r.Description)
		}
	}
	if r.Replace.Source == "" {
		errs.Addf("missing required configuration value: %s.replace.source (rule %q)", path, r.Description)
	}
	if r.Replace.Pattern == "" {
		errs.Addf("empty regex in %s.replace.pattern (rule %q) will probably not do what you think (if you actually want to match empty strings only, write `^$` to confirm your intention)", path, r.Description)
	}
	if len(r.Replace.Target) == 0 {
		errs.Addf("missing required configuration value: %s.replace.target (rule %q) needs at least one entry", path, r.Description)
	}
	return
}
