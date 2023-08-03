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
	"regexp"
	"strconv"
	"strings"
)

// ExecuteRulesOnViolation mutates the given violation by applying all matching
// rules to it.
func ExecuteRulesOnViolation(rules []Rule, v *Violation) {
	data := map[string]string{
		"kind":      v.Kind,
		"name":      v.Name,
		"namespace": v.Namespace,
		"message":   v.Message,
	}
	for key, val := range v.ObjectIdentity {
		data["object_identity."+key] = val
	}

	for _, r := range rules {
		r.execute(data)
	}

	v.Kind = data["kind"]
	v.Name = data["name"]
	v.Namespace = data["namespace"]
	v.Message = data["message"]
	for key := range v.ObjectIdentity {
		v.ObjectIdentity[key] = data["object_identity."+key]
	}
}

var placeholderRx = regexp.MustCompile(`\$[0-9][1-9]*`) // matches $0, $1, $2, etc.

func (r Rule) execute(data map[string]string) {
	//check the `match` section: can we consider applying this rule?
	for fieldName, rx := range r.Match {
		fieldValue, ok := data[fieldName]
		if !ok || !rx.MatchString(fieldValue) {
			return
		}
	}

	//check the `replace` section: can we perform a replacement?
	sourceFieldValue, ok := data[r.Replace.Source]
	if !ok {
		return
	}
	match := r.Replace.Pattern.FindStringSubmatch(sourceFieldValue)
	if match == nil {
		return
	}

	//everything matches and the rule applies - perform every requested replacement
	for fieldName, valuePattern := range r.Replace.Target {
		// in the replacement string (valuePattern), replace "$1" with match[1], "$2" with match[2], etc.
		data[fieldName] = placeholderRx.ReplaceAllStringFunc(valuePattern, func(placeholder string) string {
			idx, err := strconv.Atoi(strings.TrimPrefix(placeholder, "$")) //NOTE: idx must be positive because of placeholderRx
			if err == nil && idx < len(match) {
				return match[idx]
			} else {
				return placeholder
			}
		})
	}
}

// Process applies the configured ProcessingRules and MergingRules to this report.
func (r *Report) Process(cfg Configuration) {
	for _, rt := range r.Templates {
		for idx := range rt.Constraints {
			//In this loop, we need to address via index instead of copy-by-value
			//because the slice elements are not pointers.
			rt.Constraints[idx].process(cfg)
		}
	}
}

func (rc *ReportForConstraint) process(cfg Configuration) {
	//After GatherReport(), only rc.Violations will be filled. The goal of this
	//function is to clear out rc.Violations and fill rc.ViolationGroups instead.
	if len(rc.ViolationGroups) != 0 {
		panic("Report.Process called on a report that has already been processed")
	}

VIOLATION:
	for _, v := range rc.Violations {
		//apply processing rules first
		ExecuteRulesOnViolation(cfg.ProcessingRules, &v) //nolint:gosec // called function does not retain the pointer
		vg := ViolationGroup{Pattern: v.Cloned()}

		//apply merging rules to obtain group pattern, then try to merge into an
		//existing ViolationGroup if possible
		ExecuteRulesOnViolation(cfg.MergingRules, &vg.Pattern)
		for _, other := range rc.ViolationGroups {
			if vg.Pattern.IsEqualTo(other.Pattern) {
				other.Instances = append(other.Instances, v.DifferenceTo(vg.Pattern))
				continue VIOLATION
			}
		}

		//cannot merge -> remember new ViolationGroup
		vg.Instances = []Violation{v.DifferenceTo(vg.Pattern)}
		rc.ViolationGroups = append(rc.ViolationGroups, &vg)
	}

	rc.Violations = nil
}
