/*******************************************************************************
*
* Copyright 2021 SAP SE
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
	"encoding/json"
	"os"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sapcc/go-bits/logg"
	"github.com/sapcc/go-bits/osext"
)

var RawViolationsGauge = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "doop_raw_violations",
		Help: "number of raw violations for each check, grouped by source cluster",
	},
	[]string{"check", "source_cluster"},
)

var GroupedViolationGauge = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "doop_grouped_violations",
		Help: "Number of grouped violations for each check",
	},
	[]string{"check"},
)

var AuditAgeNewestGauge = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "doop_newest_audit_age_seconds",
		Help: "Data age for each cluster",
	},
	[]string{"cluster"},
)

var AuditAgeOldestGauge = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "doop_oldest_audit_age_seconds",
		Help: "Data age for each cluster",
	},
	[]string{"cluster"},
)

func (ui UI) Collect(ch chan<- prometheus.Metric) {
	descCh := make(chan *prometheus.Desc, 1)
	RawViolationsGauge.Describe(descCh)
	rawViolationsDesc := <-descCh
	GroupedViolationGauge.Describe(descCh)
	groupedViolationDesc := <-descCh
	AuditAgeNewestGauge.Describe(descCh)
	auditAgeNewestDesc := <-descCh
	AuditAgeOldestGauge.Describe(descCh)
	auditAgeOldestDesc := <-descCh

	showAll := false
	data, err := ui.downloader.retrieveData(showAll)
	if err != nil {
		logg.Error(err.Error())
	}

	dumpData := osext.GetenvBool("DOOP_CENTRAL_DUMP_DATA")
	if dumpData {
		dataJSON, err := json.Marshal(data)
		if err == nil {
			err = os.WriteFile("data.json", dataJSON, 0644) //nolint:gosec // file only written in debug mode
		}
		if err != nil {
			logg.Error(err.Error())
		}
	}

	for _, check := range data.AllTemplateKinds {
		groupedViolations := data.ViolationGroups[check]

		ch <- prometheus.MustNewConstMetric(
			groupedViolationDesc,
			prometheus.GaugeValue, float64(len(groupedViolations)),
			check,
		)

		violationCheckCounter := make(map[string]float64)
		for _, v := range data.AllClusters {
			violationCheckCounter[v] = 0
		}
		for _, violation := range groupedViolations {
			for _, instance := range violation.Instances {
				violationCheckCounter[instance.ClusterName]++
			}
		}
		for violation, count := range violationCheckCounter {
			ch <- prometheus.MustNewConstMetric(
				rawViolationsDesc,
				prometheus.GaugeValue, count,
				check, violation,
			)
		}
	}
	for cluster, infos := range data.ClusterInfos {
		ch <- prometheus.MustNewConstMetric(
			auditAgeNewestDesc,
			prometheus.GaugeValue, infos.NewestAuditAgeSecs,
			cluster,
		)
		ch <- prometheus.MustNewConstMetric(
			auditAgeOldestDesc,
			prometheus.GaugeValue, infos.OldestAuditAgeSecs,
			cluster,
		)
	}
}

func (ui UI) Describe(ch chan<- *prometheus.Desc) {
	RawViolationsGauge.Describe(ch)
	GroupedViolationGauge.Describe(ch)
	AuditAgeNewestGauge.Describe(ch)
	AuditAgeOldestGauge.Describe(ch)
}
