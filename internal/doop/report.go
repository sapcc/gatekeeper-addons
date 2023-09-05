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

import "sort"

// Report is the data structure that doop-analyzer produces.
type Report struct {
	ClusterIdentity map[string]string   `json:"cluster_identity"`
	Templates       []ReportForTemplate `json:"templates"`
}

// ReportForTemplate appears in type Report.
type ReportForTemplate struct {
	Kind        string                `json:"kind"`
	Constraints []ReportForConstraint `json:"constraints"`
}

// Sort sorts all lists in this report in the respective canonical order.
func (r *ReportForTemplate) Sort() {
	sort.Slice(r.Constraints, func(i, j int) bool {
		return r.Constraints[i].Name < r.Constraints[j].Name
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
	sort.Slice(r.ViolationGroups, func(i, j int) bool {
		lhs := r.ViolationGroups[i].Pattern
		rhs := r.ViolationGroups[j].Pattern
		return lhs.CompareTo(rhs) < 0
	})
	for _, vg := range r.ViolationGroups {
		sort.Slice(vg.Instances, func(i, j int) bool {
			lhs := vg.Instances[i]
			rhs := vg.Instances[j]
			return lhs.CompareTo(rhs) < 0
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
	sort.Slice(r.Templates, func(i, j int) bool {
		return r.Templates[i].Kind < r.Templates[j].Kind
	})
	for idx := range r.Templates {
		r.Templates[idx].Sort()
	}
}
