// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"encoding/json"
	"regexp"

	"github.com/sapcc/gatekeeper-addons/internal/doop"
)

// GatherReport reads all constraint templates and configs and compiles a report.
func GatherReport(ctx context.Context, cfg Configuration, cs ClientSetInterface) (doop.Report, error) {
	r := doop.Report{ClusterIdentity: cfg.ClusterIdentity}

	templates, err := cs.ListConstraintTemplates(ctx)
	if err != nil {
		return doop.Report{}, err
	}
	for _, t := range templates {
		rt, err := gatherReportForTemplate(ctx, cs, t)
		if err != nil {
			return doop.Report{}, err
		}
		if len(rt.Constraints) > 0 {
			r.Templates = append(r.Templates, rt)
		}
	}

	return r, nil
}

func gatherReportForTemplate(ctx context.Context, cs ClientSetInterface, t ConstraintTemplate) (doop.ReportForTemplate, error) {
	rt := doop.ReportForTemplate{
		Kind: t.Spec.CRD.Spec.Names.Kind,
	}

	configs, err := cs.ListConstraints(ctx, t)
	if err != nil {
		return doop.ReportForTemplate{}, err
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

func gatherReportForConstraint(c Constraint) doop.ReportForConstraint {
	cm := c.Metadata
	rc := doop.ReportForConstraint{
		Name: cm.Name,
		Metadata: doop.MetadataForConstraint{
			Severity:         cm.Labels["severity"],
			TemplateSource:   cm.Annotations["template-source"],
			ConstraintSource: cm.Annotations["constraint-source"],
			Docstring:        cm.Annotations["docstring"],
			AuditTimestamp:   c.Status.AuditTimestamp,
		},
	}

	for _, v := range c.Status.Violations {
		// extract the object identity prefix from the violation message, if any
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

		rc.Violations = append(rc.Violations, doop.Violation{
			Kind:           v.Kind,
			Name:           v.Name,
			Namespace:      v.Namespace,
			Message:        processedMessage,
			ObjectIdentity: objectIdentity,
		})
	}

	return rc
}
