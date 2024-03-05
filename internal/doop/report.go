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

package doop

import (
	"slices"
	"strings"
)

// Report is the data structure that doop-analyzer produces.
type Report struct {
	ClusterIdentity map[string]string   `json:"cluster_identity"`
	Templates       []ReportForTemplate `json:"templates"`
}

// SetClusterName sets the ClusterName field on all Violation objects in this Report.
// This is used at report loading time to prepare the report for aggregation.
// The self-return is used to shorten setup code in unit tests.
func (r Report) SetClusterName(clusterName string) Report {
	for _, t := range r.Templates {
		for _, c := range t.Constraints {
			for _, vg := range c.ViolationGroups {
				for idx, v := range vg.Instances {
					v.ClusterName = clusterName
					vg.Instances[idx] = v
				}
			}
		}
	}
	return r
}

// ReportForTemplate appears in type Report.
type ReportForTemplate struct {
	Kind        string                `json:"kind"`
	Constraints []ReportForConstraint `json:"constraints"`
}

// Sort sorts all lists in this report in the respective canonical order.
func (r *ReportForTemplate) Sort() {
	slices.SortFunc(r.Constraints, func(lhs, rhs ReportForConstraint) int {
		return strings.Compare(lhs.Name, rhs.Name)
	})
	for idx := range r.Constraints {
		r.Constraints[idx].Sort()
	}
}

// ReportForConstraint appears in type ReportForTemplate.
type ReportForConstraint struct {
	Name     string                `json:"name"`
	Metadata MetadataForConstraint `json:"metadata"`
	// Before processing, Violations is filled and ViolationGroups is nil.
	// After processing, Violations is nil and ViolationGroups is filled.
	Violations      []Violation      `json:"violations,omitempty"`
	ViolationGroups []ViolationGroup `json:"violation_groups,omitempty"`
}

// MetadataForConstraint appears in type ReportForConstraint.
type MetadataForConstraint struct {
	Severity         string `json:"severity,omitempty"`
	TemplateSource   string `json:"template_source,omitempty"`
	ConstraintSource string `json:"constraint_source,omitempty"`
	Docstring        string `json:"docstring,omitempty"`
	// AuditTimestamp is always present in type Report, but omitted in type AggregatedReport.
	AuditTimestamp string `json:"auditTimestamp,omitempty"`
}

// Sort sorts all lists in this report in the respective canonical order.
func (r *ReportForConstraint) Sort() {
	slices.SortFunc(r.ViolationGroups, func(lhs, rhs ViolationGroup) int {
		return lhs.Pattern.CompareTo(rhs.Pattern)
	})
	for _, vg := range r.ViolationGroups {
		slices.SortFunc(vg.Instances, func(lhs, rhs Violation) int {
			return lhs.CompareTo(rhs)
		})
	}
}

// AggregatedReport is the data structure that doop-api produces. It aggregates
// multiple instances of type Report from different clusters.
type AggregatedReport struct {
	ClusterIdentities map[string]map[string]string `json:"cluster_identities"`
	Templates         []ReportForTemplate          `json:"templates"`
}

// Sort sorts all lists in this report in the respective canonical order.
func (r *AggregatedReport) Sort() {
	slices.SortFunc(r.Templates, func(lhs, rhs ReportForTemplate) int {
		return strings.Compare(lhs.Kind, rhs.Kind)
	})
	for idx := range r.Templates {
		r.Templates[idx].Sort()
	}
}
