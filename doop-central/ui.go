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
	"regexp"
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
		"titlecase":  titlecase,
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
		AllClusterLayers []string
		AllClusterTypes  []string
		ClusterInfos     map[string]ClusterInfo
		AllTemplateKinds []string
		ViolationGroups  map[string][]*ViolationGroup
	}{
		ClusterInfos:    make(map[string]ClusterInfo),
		ViolationGroups: make(map[string][]*ViolationGroup),
	}

	for clusterName, report := range reports {
		data.AllClusters = append(data.AllClusters, clusterName)
		data.AllClusterLayers = append(data.AllClusterLayers, report.Identity.Layer)
		data.AllClusterTypes = append(data.AllClusterTypes, report.Identity.Type)
		data.ClusterInfos[clusterName] = report.ToClusterInfo()
		report.GroupViolationsInto(data.ViolationGroups, clusterName)
	}

	sort.Strings(data.AllClusters)
	data.AllClusterLayers = sortAndDedupStrings(data.AllClusterLayers)
	data.AllClusterTypes = sortAndDedupStrings(data.AllClusterTypes)
	for kind, violationGroups := range data.ViolationGroups {
		data.AllTemplateKinds = append(data.AllTemplateKinds, kind)
		sortViolationGroups(violationGroups)
	}
	sort.Strings(data.AllTemplateKinds)

	w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'self' 'unsafe-inline'")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	err = pageTemplate.Execute(w, data)
	if err != nil {
		logg.Error("while rendering index.html: %s", err.Error())
	}
}

func titlecase(in string) string {
	if in == "qa" {
		return "QA"
	}
	return strings.Title(in)
}

func jsonIndent(in []byte) string {
	var buf bytes.Buffer
	json.Indent(&buf, in, "", "  ")
	return buf.String()
}

func sortAndDedupStrings(vals []string) []string {
	isVal := make(map[string]bool)
	for _, val := range vals {
		isVal[val] = true
	}
	result := make([]string, 0, len(isVal))
	for val := range isVal {
		result = append(result, val)
	}
	sort.Strings(result)
	return result
}

////////////////////////////////////////////////////////////////////////////////
// report datatypes and structured data for HTML template

//ClusterInfo contains health information for the Gatekeeper in a certain cluster.
type ClusterInfo struct {
	Layer               string
	Type                string
	OldestAuditAgeSecs  float64
	OldestAuditCSSClass string
	NewestAuditAgeSecs  float64
	NewestAuditCSSClass string
}

//ToClusterInfo generates the ClusterInfo for this Report.
func (r Report) ToClusterInfo() ClusterInfo {
	now := time.Now()
	info := ClusterInfo{
		Layer: r.Identity.Layer,
		Type:  r.Identity.Type,
	}
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

//ViolationGroup is a group of mostly identical violations across clusters and
//across objects.
type ViolationGroup struct {
	//object metadata
	Kind        string
	NamePattern string
	Namespace   string
	//violation details
	Message   string
	Instances []ViolationInstance
}

//ViolationInstance appears in type ViolationGroup.
type ViolationInstance struct {
	ClusterName string
	Name        string
}

var (
	helm2ReleaseNameRx        = regexp.MustCompile(`^(.*)\.v\d+$`)
	helm3ReleaseNameRx        = regexp.MustCompile(`^sh\.helm\.release\.v1\.(.*)\.v\d+$`)
	generatedPodNameRx        = regexp.MustCompile(`^(.+)-[0-9a-z]{5}$`)
	generatedReplicasetNameRx = regexp.MustCompile(`^(.+)-[0-9a-f]{8,10}(-\.\.\.)?$`)
)

//NewViolationGroup creates a fresh group for a reported violation.
func NewViolationGroup(report ViolationReport, clusterName string) ViolationGroup {
	//special handling for Helm 2 releases
	if report.Kind == "ConfigMap" && report.Namespace == "kube-system" {
		match := helm2ReleaseNameRx.FindStringSubmatch(report.Name)
		if match != nil {
			return ViolationGroup{
				Kind:        "Helm 2 release",
				NamePattern: match[1],
				Namespace:   "",
				Message:     report.Message,
				Instances: []ViolationInstance{{
					ClusterName: clusterName,
					Name:        report.Name,
				}},
			}
		}
	}

	//special handling for Helm 3 releases
	if report.Kind == "Secret" {
		match := helm3ReleaseNameRx.FindStringSubmatch(report.Name)
		if match != nil {
			return ViolationGroup{
				Kind:        "Helm 3 release",
				NamePattern: match[1],
				Namespace:   report.Namespace,
				Message:     report.Message,
				Instances: []ViolationInstance{{
					ClusterName: clusterName,
					Name:        report.Name,
				}},
			}
		}
	}

	//normal handling for Kubernetes objects: use generated name patterns for grouping
	namePattern := report.Name
	if report.Kind == "Pod" {
		match := generatedPodNameRx.FindStringSubmatch(namePattern)
		if match != nil {
			namePattern = match[1] + "-..."
		}
	}
	if report.Kind == "Pod" || report.Kind == "ReplicaSet" {
		match := generatedReplicasetNameRx.FindStringSubmatch(namePattern)
		if match != nil {
			namePattern = match[1] + "-..." + match[2]
		}
	}

	return ViolationGroup{
		Kind:        report.Kind,
		NamePattern: namePattern,
		Namespace:   report.Namespace,
		Message:     report.Message,
		Instances: []ViolationInstance{{
			ClusterName: clusterName,
			Name:        report.Name,
		}},
	}
}

//CanMergeWith checks if both ViolationGroups are semantically identical and
//can be merged.
func (vg ViolationGroup) CanMergeWith(other ViolationGroup) bool {
	return vg.Kind == other.Kind && vg.Namespace == other.Namespace &&
		vg.NamePattern == other.NamePattern && vg.Message == other.Message
}

//GroupViolationsInto processes the violations in this report into
//ViolationGroups, sorted by template kind.
func (r Report) GroupViolationsInto(violationGroups map[string][]*ViolationGroup, clusterName string) {
	for _, rt := range r.Templates {
		for _, rc := range rt.Configs {
		VIOLATION:
			for _, rv := range rc.Violations {
				//start with a fresh violation group for this violation...
				vgNew := NewViolationGroup(rv, clusterName)

				//...but prefer to merge it with an existing group
				for _, vgOld := range violationGroups[rt.Kind] {
					if vgOld.CanMergeWith(vgNew) {
						vgOld.Instances = append(vgOld.Instances, vgNew.Instances...)
						continue VIOLATION
					}
				}

				//otherwise it gets inserted on its own
				violationGroups[rt.Kind] = append(violationGroups[rt.Kind], &vgNew)
			}
		}
	}
}

func sortViolationGroups(groups []*ViolationGroup) {
	for _, group := range groups {
		sortViolationInstances(group.Instances)
	}
	sort.Slice(groups, func(i, j int) bool {
		lhs := groups[i]
		rhs := groups[j]
		if lhs.Kind != rhs.Kind {
			return lhs.Kind < rhs.Kind
		}
		if lhs.Namespace != rhs.Namespace {
			return lhs.Namespace < rhs.Namespace
		}
		if lhs.NamePattern != rhs.NamePattern {
			return lhs.NamePattern < rhs.NamePattern
		}
		return lhs.Message < rhs.Message
	})
}

func sortViolationInstances(instances []ViolationInstance) {
	sort.Slice(instances, func(i, j int) bool {
		lhs := instances[i]
		rhs := instances[j]
		if lhs.ClusterName != rhs.ClusterName {
			return lhs.ClusterName < rhs.ClusterName
		}
		return lhs.Name < rhs.Name
	})
}
