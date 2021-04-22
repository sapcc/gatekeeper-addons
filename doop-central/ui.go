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
	"bytes"
	_ "embed"
	"encoding/json"
	"net/http"
	"sort"
	"strings"
	"text/template"
	"time"

	"github.com/sapcc/go-bits/logg"
)

//go:embed index.html.tpl
var pageTemplateStr string

var (
	pageTemplate = template.Must(template.New("index.html").Funcs(funcMap).Parse(pageTemplateStr))
	funcMap      = template.FuncMap{
		"jsonIndent": jsonIndent,
	}
)

////////////////////////////////////////////////////////////////////////////////
// type UI

//UI provides the business logic for rendering the web dashboard.
type UI struct {
	d *Downloader
}

//RenderMainPage is a http.HandleFunc for `GET /`.
func (ui UI) RenderMainPage(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" || (r.Method != "GET" && r.Method != "HEAD") {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	reports, err := ui.d.GetReports()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := struct {
		AllClusters      []string
		AllClusterGroups []string
		ClustersByGroup  map[string][]string
		ClusterInfos     map[string]ClusterInfo
		Reports          map[string][]byte //TODO remove (only used for debug display)
	}{
		ClustersByGroup: make(map[string][]string),
		ClusterInfos:    make(map[string]ClusterInfo),
		Reports:         make(map[string][]byte),
	}

	for clusterName, report := range reports {
		data.AllClusters = append(data.AllClusters, clusterName)
		clusterGroup := clusterGroupOf(clusterName)
		data.ClustersByGroup[clusterGroup] = append(data.ClustersByGroup[clusterGroup], clusterName)
		data.ClusterInfos[clusterName] = report.ToClusterInfo()

		reportBytes, _ := json.Marshal(report)
		data.Reports[clusterName] = reportBytes
	}

	sort.Strings(data.AllClusters)
	for clusterGroup, clusterNames := range data.ClustersByGroup {
		sort.Strings(clusterNames)
		data.AllClusterGroups = append(data.AllClusterGroups, clusterGroup)
	}
	sort.Strings(data.AllClusterGroups)

	w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'self' 'unsafe-inline'")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	err = pageTemplate.Execute(w, data)
	if err != nil {
		logg.Error("while rendering index.html: %s", err.Error())
	}
}

func jsonIndent(in []byte) string {
	var buf bytes.Buffer
	json.Indent(&buf, in, "", "  ")
	return buf.String()
}

////////////////////////////////////////////////////////////////////////////////
// report datatypes and structured data for HTML template

func clusterGroupOf(clusterName string) string {
	for _, prefix := range []string{"a-", "k-", "s-", "v-"} {
		if strings.HasPrefix(clusterName, prefix) {
			return prefix
		}
	}
	return ""
}

//ClusterInfo contains health information for the Gatekeeper in a certain cluster.
type ClusterInfo struct {
	OldestAuditAgeSecs  float64
	OldestAuditCSSClass string
	NewestAuditAgeSecs  float64
	NewestAuditCSSClass string
}

//ToClusterInfo generates the ClusterInfo for this Report.
func (r Report) ToClusterInfo() ClusterInfo {
	now := time.Now()
	var info ClusterInfo
	for _, rt := range r.Templates {
		for _, rc := range rt.Configs {
			auditAgeSecs := now.Sub(rc.AuditAt).Seconds()
			if info.OldestAuditAgeSecs == 0 || info.OldestAuditAgeSecs < auditAgeSecs {
				info.OldestAuditAgeSecs = auditAgeSecs
			}
			if info.NewestAuditAgeSecs == 0 || info.NewestAuditAgeSecs > auditAgeSecs {
				info.NewestAuditAgeSecs = auditAgeSecs
			}
		}
	}

	info.OldestAuditCSSClass = cssClassForAge(info.OldestAuditAgeSecs)
	info.NewestAuditCSSClass = cssClassForAge(info.NewestAuditAgeSecs)
	return info
}

func cssClassForAge(ageSecs float64) string {
	if ageSecs >= 900 {
		return "value-danger"
	}
	if ageSecs >= 300 {
		return "value-warning"
	}
	return "value-ok"
}
