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
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sapcc/go-bits/logg"

	"github.com/sapcc/gatekeeper-addons/internal/doop"
)

// MetricCollector is a prometheus.Collector for all metrics provided by this application.
type MetricCollector struct {
	downloader             *Downloader
	objectIdentityKeys     []string
	rawViolationsGauge     *prometheus.GaugeVec
	groupedViolationsGauge *prometheus.GaugeVec
	auditAgeOldestGauge    *prometheus.GaugeVec
}

// NewMetricCollector initializes a MetricCollector.
func NewMetricCollector(downloader *Downloader) *MetricCollector {
	objectIdentityKeys := strings.Fields(os.Getenv("DOOP_API_OBJECT_IDENTITY_LABELS"))

	//the given key names may not be suitable as label names because of the restricted label name grammar
	//-> sanitize all non-alphanumeric characters into underscores for the label names
	objectIdentityLabels := make([]string, len(objectIdentityKeys))
	rx := regexp.MustCompile(`[^a-zA-Z0-9]`)
	for idx, key := range objectIdentityKeys {
		objectIdentityLabels[idx] = rx.ReplaceAllString(key, "_")
	}

	return &MetricCollector{
		downloader:         downloader,
		objectIdentityKeys: objectIdentityKeys,
		rawViolationsGauge: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "doop_raw_violations",
				Help: "Number of raw violations, grouped by constraint, source cluster and selected object identity labels.",
			},
			append([]string{"cluster", "template_kind", "constraint_name", "severity"}, objectIdentityLabels...),
		),
		groupedViolationsGauge: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "doop_grouped_violations",
				Help: "Number of violation groups, grouped by constraint, source cluster and selected object identity labels.",
			},
			append([]string{"template_kind", "constraint_name", "severity"}, objectIdentityLabels...),
		),
		auditAgeOldestGauge: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "doop_oldest_audit_age_seconds",
				Help: "Data age for each source cluster.",
			},
			[]string{"cluster"},
		),
	}
}

// Describe implements the prometheus.Collector interface.
func (mc *MetricCollector) Describe(ch chan<- *prometheus.Desc) {
	mc.rawViolationsGauge.Describe(ch)
	mc.groupedViolationsGauge.Describe(ch)
	mc.auditAgeOldestGauge.Describe(ch)
}

// Collect implements the prometheus.Collector interface.
func (mc *MetricCollector) Collect(ch chan<- prometheus.Metric) {
	descCh := make(chan *prometheus.Desc, 1)
	mc.rawViolationsGauge.Describe(descCh)
	rawViolationsDesc := <-descCh
	mc.groupedViolationsGauge.Describe(descCh)
	groupedViolationsDesc := <-descCh
	mc.auditAgeOldestGauge.Describe(descCh)
	auditAgeOldestDesc := <-descCh

	//using the individual reports, we can immediately calculate the audit age
	reports, err := mc.downloader.GetReports()
	if err != nil {
		logg.Error("could not download reports for metric computation: %s", err.Error())
	}
	for clusterName, report := range reports {
		ch <- prometheus.MustNewConstMetric(
			auditAgeOldestDesc,
			prometheus.GaugeValue, oldestAuditAgeForClusterReport(clusterName, report),
			clusterName,
		)
	}

	//counting violation groups requires an aggregated report
	fullReport := AggregateReports(reports, BuildFilterSet(url.Values{}))
	for _, rt := range fullReport.Templates {
		//there may be multiple constraints with the same name if metadata differs;
		//to avoid reporting metrics with the same labelsets multiple times,
		//we need to group by constraint name first
		reportsByConstraintName := make(map[string][]doop.ReportForConstraint)
		for _, rc := range rt.Constraints {
			reportsByConstraintName[rc.Name] = append(reportsByConstraintName[rc.Name], rc)
		}
		for constraintName, rcs := range reportsByConstraintName {
			countViolationsForConstraint(rt.Kind, constraintName, rcs, mc.objectIdentityKeys, rawViolationsDesc, groupedViolationsDesc, ch)
		}
	}
}

func oldestAuditAgeForClusterReport(clusterName string, report doop.Report) float64 {
	result := -1.0
	for _, rt := range report.Templates {
		for _, rc := range rt.Constraints {
			auditTimeStr := rc.Metadata.AuditTimestamp
			if auditTimeStr == "" {
				continue
			}

			auditTime, err := time.Parse(time.RFC3339, auditTimeStr)
			if err != nil {
				logg.Error("cannot parse audit timestamp %q for cluster %s: %s",
					auditTimeStr, clusterName, err.Error())
				//to ensure that the error is noticed, report a very old timestamp that likely triggers alerts
				return 0
			}

			auditAge := time.Since(auditTime).Seconds()
			if result == -1 || result < auditAge {
				result = auditAge
			}
		}
	}
	return result
}

func countViolationsForConstraint(templateKind, constraintName string, rcs []doop.ReportForConstraint, oidKeys []string, rawViolationsDesc, groupedViolationsDesc *prometheus.Desc, ch chan<- prometheus.Metric) {
	//NOTE: This function uses "oid" as an abbreviation for "object identity".

	//First map key is severity, second map is the relevant oid values, third map key is the cluster name.
	//Since we do not know how many oid keys we will have in advance,
	//we merge them all together into one string with "\0" as a field separator.
	rawCounts := make(map[string]map[string]map[string]int)
	//No cluster name here, only the severity and relevant oid values.
	groupedCounts := make(map[string]map[string]int)

	//count violations and violation groups
	for _, rc := range rcs {
		for _, vg := range rc.ViolationGroups {
			oidValues := make([]string, len(oidKeys))
			for idx, key := range oidKeys {
				oidValues[idx] = vg.Pattern.ObjectIdentity[key]
			}
			oidValuesStr := strings.Join(oidValues, "\000")
			severity := rc.Metadata.Severity

			if groupedCounts[severity] == nil {
				groupedCounts[severity] = make(map[string]int)
			}
			groupedCounts[severity][oidValuesStr]++

			if rawCounts[severity] == nil {
				rawCounts[severity] = make(map[string]map[string]int)
			}
			if rawCounts[severity][oidValuesStr] == nil {
				rawCounts[severity][oidValuesStr] = make(map[string]int)
			}
			for _, v := range vg.Instances {
				rawCounts[severity][oidValuesStr][v.ClusterName]++
			}
		}
	}

	//emit metrics
	for severity, subcounts := range groupedCounts {
		for oidValuesStr, count := range subcounts {
			labels := make([]string, 3, 3+len(oidKeys))
			labels[0] = templateKind
			labels[1] = constraintName
			labels[2] = severity
			labels = append(labels, strings.Split(oidValuesStr, "\000")...)
			ch <- prometheus.MustNewConstMetric(
				groupedViolationsDesc,
				prometheus.GaugeValue, float64(count),
				labels...,
			)
		}
	}

	for severity, subcounts := range rawCounts {
		for oidValuesStr, subsubcounts := range subcounts {
			labels := make([]string, 4, 4+len(oidKeys))
			labels[1] = templateKind
			labels[2] = constraintName
			labels[3] = severity
			labels = append(labels, strings.Split(oidValuesStr, "\000")...)
			for clusterName, count := range subsubcounts {
				labels[0] = clusterName
				ch <- prometheus.MustNewConstMetric(
					rawViolationsDesc,
					prometheus.GaugeValue, float64(count),
					labels...,
				)
			}
		}
	}
}
