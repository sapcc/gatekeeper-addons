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
	"context"
	"encoding/json"
	"regexp"
)

// Report is the data structure that we write into our report file in Swift.
type Report struct {
	ClusterIdentity map[string]string   `json:"cluster_identity"`
	Templates       []ReportForTemplate `json:"templates"`
}

// ReportForTemplate appears in type Report.
type ReportForTemplate struct {
	Kind        string                `json:"kind"`
	Constraints []ReportForConstraint `json:"constraints"`
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
	AuditTimestamp   string `json:"auditTimestamp"`
}

// ViolationGroup appears in type ReportForConstraint.
type ViolationGroup struct {
	Pattern   Violation   `json:"pattern"`
	Instances []Violation `json:"instances"`
}

// Violation describes a single policy violation, or the common pattern within a
// ViolationGroup.
type Violation struct {
	Kind           string            `json:"kind"`
	Name           string            `json:"name"`
	Namespace      string            `json:"namespace"`
	Message        string            `json:"message"`
	ObjectIdentity map[string]string `json:"object_identity,omitempty"`
}

// GatherReport reads all constraint templates and configs and compiles a report.
func GatherReport(ctx context.Context, cfg Configuration, cs ClientSetInterface) (Report, error) {
	r := Report{ClusterIdentity: cfg.ClusterIdentity}

	templates, err := cs.ListConstraintTemplates(ctx)
	if err != nil {
		return Report{}, err
	}
	for _, t := range templates {
		rt, err := gatherReportForTemplate(ctx, cs, t)
		if err != nil {
			return Report{}, err
		}
		if len(rt.Constraints) > 0 {
			r.Templates = append(r.Templates, rt)
		}
	}

	return r, nil
}

func gatherReportForTemplate(ctx context.Context, cs ClientSetInterface, t ConstraintTemplate) (ReportForTemplate, error) {
	rt := ReportForTemplate{
		Kind: t.Spec.CRD.Spec.Names.Kind,
	}

	configs, err := cs.ListConstraints(ctx, t)
	if err != nil {
		return ReportForTemplate{}, err
	}
	for _, c := range configs {
		rc := gatherReportForConstraint(c)
		if len(rc.Violations) > 0 {
			rt.Constraints = append(rt.Constraints, rc)
		}
	}

	return rt, nil
}

var objectIdentityRx = regexp.MustCompile(`^(\{.*?\})\s*>>\s*(.*)$`)

func gatherReportForConstraint(c Constraint) ReportForConstraint {
	cm := c.Metadata
	rc := ReportForConstraint{
		Name: cm.Name,
		Metadata: MetadataForConstraint{
			Severity:         cm.Labels["severity"],
			TemplateSource:   cm.Annotations["template-source"],
			ConstraintSource: cm.Annotations["constraint-source"],
			Docstring:        cm.Annotations["docstring"],
			AuditTimestamp:   c.Status.AuditTimestamp,
		},
	}

	for _, v := range c.Status.Violations {
		//extract the object identity prefix from the violation message, if any
		processedMessage := v.Message
		var objectIdentity map[string]string
		match := objectIdentityRx.FindStringSubmatch(processedMessage)
		if match != nil {
			err := json.Unmarshal([]byte(match[1]), &objectIdentity)
			if err == nil {
				processedMessage = match[2]
			} else {
				objectIdentity = nil
			}
		}

		rc.Violations = append(rc.Violations, Violation{
			Kind:           v.Kind,
			Name:           v.Name,
			Namespace:      v.Namespace,
			Message:        processedMessage,
			ObjectIdentity: objectIdentity,
		})
	}

	return rc
}
