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
	"slices"

	"github.com/sapcc/gatekeeper-addons/internal/doop"
)

// AggregateReports assembles a set of individual reports into an AggregatedReport.
func AggregateReports(reports map[string]doop.Report, f FilterSet) doop.AggregatedReport {
	target := doop.AggregatedReport{
		ClusterIdentities: make(map[string]map[string]string),
	}

	for clusterName, clusterReport := range reports {
		visitClusterReport(&target, clusterReport, clusterName, f)
	}

	return target
}

// NOTE on the general implementation style: This uses a variant of the visitor pattern.
// For each level that is processed, we have one function to avoid nesting loops too deeply within any one function.
// To keep the output compact, we only add objects to lists if the lower levels of the call stack actually insert data.
// We generally try to avoid heap allocations as much as possible for temporary objects because the nested loops make the inner functions very hot.

func visitClusterReport(target *doop.AggregatedReport, clusterReport doop.Report, clusterName string, f FilterSet) {
	if !f.MatchClusterIdentity(clusterReport.ClusterIdentity) {
		return
	}

	target.ClusterIdentities[clusterName] = clusterReport.ClusterIdentity
	for _, tr := range clusterReport.Templates {
		visitTemplateReport(target, tr, f)
	}
}

func visitTemplateReport(target *doop.AggregatedReport, tr doop.ReportForTemplate, f FilterSet) {
	if !f.MatchTemplateKind(tr.Kind) {
		return
	}

	// try to merge into existing ReportForTemplate
	for idx, candidate := range target.Templates {
		if candidate.Kind == tr.Kind {
			for _, cr := range tr.Constraints {
				visitConstraintReport(&target.Templates[idx], cr, f)
			}
			return
		}
	}

	// otherwise try to start a new ReportForTemplate
	newReport := doop.ReportForTemplate{
		Kind: tr.Kind,
	}
	for _, cr := range tr.Constraints {
		visitConstraintReport(&newReport, cr, f)
	}
	if len(newReport.Constraints) > 0 {
		target.Templates = append(target.Templates, newReport)
	}
}

func visitConstraintReport(target *doop.ReportForTemplate, cr doop.ReportForConstraint, f FilterSet) {
	if !f.MatchConstraintName(cr.Name) {
		return
	}
	if !f.MatchSeverity(cr.Metadata.Severity) {
		return
	}

	// the Metadata.AuditTimestamp field is only used to generate Prometheus metrics; it is not aggregated
	metadata := cr.Metadata
	metadata.AuditTimestamp = ""

	// try to merge into existing ReportForConstraint
	for idx, candidate := range target.Constraints {
		if candidate.Name == cr.Name && candidate.Metadata == metadata {
			for _, vg := range cr.ViolationGroups {
				visitViolationGroup(&target.Constraints[idx], vg, f)
			}
			return
		}
	}

	// otherwise try to start a new ReportForConstraint
	newReport := doop.ReportForConstraint{
		Name:     cr.Name,
		Metadata: metadata,
	}
	for _, vg := range cr.ViolationGroups {
		visitViolationGroup(&newReport, vg, f)
	}
	if len(newReport.ViolationGroups) > 0 {
		target.Constraints = append(target.Constraints, newReport)
	}
}

func visitViolationGroup(target *doop.ReportForConstraint, vg doop.ViolationGroup, f FilterSet) {
	if !f.MatchObjectIdentity(vg.Pattern.ObjectIdentity) {
		return
	}

	// try to merge into existing ViolationGroup
	for idx, candidate := range target.ViolationGroups {
		if candidate.Pattern.IsEqualTo(vg.Pattern) {
			target.ViolationGroups[idx].Instances = append(target.ViolationGroups[idx].Instances, vg.Instances...)
			return
		}
	}

	// otherwise start a new ViolationGroup
	target.ViolationGroups = append(target.ViolationGroups, doop.ViolationGroup{
		Pattern:   vg.Pattern.Cloned(),
		Instances: slices.Clone(vg.Instances),
	})
}
