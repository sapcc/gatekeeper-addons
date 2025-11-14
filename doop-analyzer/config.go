// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/sapcc/go-bits/errext"
	"github.com/sapcc/go-bits/regexpext"
)

// Configuration contains the contents of the config file.
type Configuration struct {
	ClusterIdentity map[string]string `json:"cluster_identity"`
	Kubernetes      struct {
		KubeconfigPath string `json:"kubeconfig"`
		Context        string `json:"context"`
	} `json:"kubernetes"`
	Metrics struct {
		ListenAddress string `json:"listen_address"`
	} `json:"metrics"`
	MergingRules    []Rule             `json:"merging_rules"`
	ProcessingRules []Rule             `json:"processing_rules"`
	Swift           SwiftConfiguration `json:"swift"`
}

// Rule is a rule that can appear in `processing_rules` or `merging_rules`.
type Rule struct {
	Description string                             `json:"description"`
	Match       map[string]regexpext.BoundedRegexp `json:"match"`
	Replace     ReplaceRule                        `json:"replace"`
}

// ReplaceRule appears in type Rule.
type ReplaceRule struct {
	Source  string                  `json:"source"`
	Pattern regexpext.BoundedRegexp `json:"pattern"`
	Target  map[string]string       `json:"target"`
}

// ReadConfiguration reads the config file at the given path.
func ReadConfiguration(configPath string) (Configuration, error) {
	buf, err := os.ReadFile(configPath)
	if err != nil {
		return Configuration{}, err
	}

	var cfg Configuration
	err = json.Unmarshal(buf, &cfg)
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
