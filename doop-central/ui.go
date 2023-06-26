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
	_ "embed"
	"fmt"
	"html/template"
	"net/http"
	"path"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/sapcc/go-bits/logg"
	"github.com/sapcc/go-bits/respondwith"
)

//go:embed index.html.tpl
var pageTemplateStr string

var (
	pageTemplate = template.Must(template.New("index.html").Funcs(funcMap).Parse(pageTemplateStr))
	funcMap      = template.FuncMap{
		"titlecase":          titlecase,
		"markupPlaceholders": markupPlaceholders,
		"basenameAndTrim":    basenameAndTrim,
	}
)

////////////////////////////////////////////////////////////////////////////////
// type UI

// UI provides the business logic for rendering the web dashboard.
type UI struct {
	downloader *Downloader
	docstrings map[string]template.HTML
}

// AddTo implements the httpapi.API interface.
func (ui UI) AddTo(r *mux.Router) {
	r.Methods("HEAD", "GET").Path("/").HandlerFunc(ui.renderMainPage)
	r.Methods("HEAD", "GET").Path("/all").HandlerFunc(ui.renderMainPage)
	r.Methods("GET").Path("/json").HandlerFunc(ui.renderAPI)
	r.Methods("GET").Path("/json/all").HandlerFunc(ui.renderAPI)
}

// renderAPI is a http.HandleFunc for `GET /json` and `GET /json/all`.
func (ui UI) renderAPI(w http.ResponseWriter, r *http.Request) {
	showAll := r.URL.Path == "/json/all"
	data, err := ui.downloader.retrieveData(showAll)
	if respondwith.ErrorText(w, err) {
		return
	}
	respondwith.JSON(w, http.StatusOK, data.APIData)
}

// renderMainPage is a http.HandleFunc for `GET /` and `GET /all`.
func (ui UI) renderMainPage(w http.ResponseWriter, r *http.Request) {
	showAll := r.URL.Path == "/all"
	data, err := ui.downloader.retrieveData(showAll)
	data.Docstrings = ui.docstrings
	if respondwith.ErrorText(w, err) {
		return
	}

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
	return strings.Title(in) //nolint:staticcheck //ignore SA1019: this function is still good for ASCII-only inputs
}

func basenameAndTrim(prefix, text, suffix string) string {
	text = path.Base(text)
	text = strings.TrimPrefix(text, prefix)
	text = strings.TrimSuffix(text, suffix)
	return text
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

var placeholderRx = regexp.MustCompile(`<(variable|index|cluster|region)>`)

func markupPlaceholders(in string) template.HTML {
	//When grouping violations together (see below), we do certain string
	//replacements in the object name, object namespace and violation message in
	//order to enable merge aggressive grouping. This template function replaces
	//the pseudo-variables used therein with some proper HTML markup to make the
	//pseudo-variables stand out better on screen.
	out := ""
	for {
		loc := placeholderRx.FindStringSubmatchIndex(in)
		if loc == nil {
			break
		}
		out += template.HTMLEscapeString(in[:loc[0]])
		out += fmt.Sprintf(`<span class="collation-variable">%s</span>`, in[loc[2]:loc[3]])
		in = in[loc[1]:]
	}

	return template.HTML(out + template.HTMLEscapeString(in)) //nolint:gosec // no xss injection possible due to prior checks of input
}

////////////////////////////////////////////////////////////////////////////////
// report datatypes and structured data for API and HTML template

// ClusterInfo contains health information for the Gatekeeper in a certain cluster.
type ClusterInfo struct {
	Layer        string  `json:"layer"`
	Type         string  `json:"type"`
	AuditAgeSecs float64 `json:"audit_age_secs"`
	AuditStatus  string  `json:"audit_status"`
}

// ToClusterInfo generates the ClusterInfo for this Report.
func (r Report) ToClusterInfo() ClusterInfo {
	now := time.Now()
	info := ClusterInfo{
		Layer: r.Identity.Layer,
		Type:  r.Identity.Type,
	}
	for _, rt := range r.Templates {
		for _, rc := range rt.Configs {
			auditAgeSecs := now.Sub(rc.AuditAt).Seconds()
			if info.AuditAgeSecs == 0 || info.AuditAgeSecs < auditAgeSecs {
				info.AuditAgeSecs = auditAgeSecs
			}
		}
	}

	info.AuditStatus = "ok"
	if info.AuditAgeSecs >= 300 {
		info.AuditStatus = "warning"
	}
	if info.AuditAgeSecs >= 900 {
		info.AuditStatus = "danger"
	}
	return info
}

// KindInfo contains structured metadata for a certain kind of constraint template.
type KindInfo struct {
	TemplateSources   []string `json:"template_source_urls"`
	ConstraintSources []string `json:"constraint_source_urls"`
}

func (k *KindInfo) addTemplateSource(src string) {
	for _, existing := range k.TemplateSources {
		if existing == src {
			return
		}
	}
	k.TemplateSources = append(k.TemplateSources, src)
	sort.Strings(k.TemplateSources)
}

func (k *KindInfo) addConstraintSource(src string) {
	for _, existing := range k.ConstraintSources {
		if existing == src {
			return
		}
	}
	k.ConstraintSources = append(k.ConstraintSources, src)
	sort.Strings(k.ConstraintSources)
}

// ViolationGroup is a group of mostly identical violations across clusters and
// across objects.
type ViolationGroup struct {
	//object metadata
	Kind              string `json:"kind"`
	NamePattern       string `json:"name_pattern"`
	Namespace         string `json:"namespace"`
	SupportGroupLabel string `json:"support_group"`
	ServiceLabel      string `json:"service"`
	//violation details
	Message   string              `json:"msg_pattern"`
	Instances []ViolationInstance `json:"instances"`
}

// ViolationInstance appears in type ViolationGroup.
type ViolationInstance struct {
	ClusterName string `json:"cluster"`
	Name        string `json:"name"`
}

var (
	supportLabelsRx                 = regexp.MustCompile(`^support-group=([a-z0-9-]+),service=([a-z0-9-]+):\s*(.*)$`)
	helm3ReleaseNameRx              = regexp.MustCompile(`^sh\.helm\.release\.v1\.(.*)(\.v\d+)$`)
	generatedNamespaceNameRx        = regexp.MustCompile(`^[0-9a-f]{32}$`)
	generatedKubernikusUUIDRx       = regexp.MustCompile(`\b[0-9a-f]{32}\b`)
	generatedPodNameRx              = regexp.MustCompile(`^(.+)-[0-9a-z]{5}$`)
	generatedReplicasetNameRx       = regexp.MustCompile(`^(.+)-[0-9a-f]{8,10}(-<variable>)?$`)
	generatedPodInStatefulsetNameRx = regexp.MustCompile(`^(.+)-[0-9]{1,2}$`)
	generatedOverlongPodNameRx      = regexp.MustCompile(`^(.+)-[0-9a-f]{3,10}[0-9a-z]{5}$`)
	generatedCloudShellPodNameRx    = regexp.MustCompile(`^shell-(?:c[0-9]{7}|[di][0-9]{6})$`)
	regionNameInClusterNameRx       = regexp.MustCompile(`^(?:[a-z]-)?(.*)$`)
)

// NewViolationGroup creates a fresh group for a reported violation.
func NewViolationGroup(report ViolationReport, clusterName string) ViolationGroup {
	computedKind := report.Kind
	namePattern := report.Name
	namespacePattern := report.Namespace
	messagePattern := report.Message

	//extract the "support-group=XXX,service=YYY: " prefix
	supportGroupLabel, serviceLabel := "none", "none"
	match := supportLabelsRx.FindStringSubmatch(messagePattern)
	if match != nil {
		supportGroupLabel, serviceLabel, messagePattern = match[1], match[2], match[3]
	}

	//for now, we ignore the "support-group=XXX,service=YYY: " prefixes entirely;
	//later this will be changed once adoption is far enough to restructure the
	//UI around these categories
	messagePattern = supportLabelsRx.ReplaceAllString(messagePattern, "")

	//special handling for Helm 3 releases
	if report.Kind == "Secret" {
		match := helm3ReleaseNameRx.FindStringSubmatch(report.Name)
		if match != nil {
			computedKind = "Helm 3 release"
			namePattern = match[1]
			report.Name = match[1] + match[2]
		}
	}

	//normal handling for Kubernetes objects: use generated name patterns for grouping
	if report.Kind == "Pod" {
		match := generatedPodNameRx.FindStringSubmatch(namePattern)
		if match != nil {
			namePattern = match[1] + "-<variable>"
		}
		match = generatedPodInStatefulsetNameRx.FindStringSubmatch(namePattern)
		if match != nil {
			namePattern = match[1] + "-<index>"
		}
	}
	if report.Kind == "Pod" || report.Kind == "ReplicaSet" {
		match := generatedReplicasetNameRx.FindStringSubmatch(namePattern)
		if match != nil {
			namePattern = match[1] + "-<variable>" + match[2]
		}
	}

	//when the name is incredibly long, the autogenerated replicaset name will
	//have to be truncated before adding the `[0-9a-z]{5}` prefix for the pod, so
	//both will appear merged into each other
	if report.Kind == "Pod" && len(report.Name) == 63 {
		match := generatedOverlongPodNameRx.FindStringSubmatch(namePattern)
		if match != nil {
			namePattern = match[1] + "-<variable>"
		}
	}

	//special case for grafana-operator: merge violations in different namespaces
	//if the namespaces are all autogenerated by grafana-operator
	if strings.HasPrefix(namePattern, "grafana-") && generatedNamespaceNameRx.MatchString(namespacePattern) {
		namespacePattern = "<variable>"
	}

	//special case for Kubernikus: merge violations for Kubernikus clusters with the same name that only differ in UUID
	namePattern = generatedKubernikusUUIDRx.ReplaceAllString(namePattern, "<variable>")
	messagePattern = generatedKubernikusUUIDRx.ReplaceAllString(messagePattern, "<variable>")

	//special case for cloud-shell: merge violations for pods managed by cloud-shell
	if report.Namespace == "cloud-shell" {
		match := generatedCloudShellPodNameRx.FindStringSubmatch(namePattern)
		if match != nil {
			namePattern = "shell-<variable>"
		}
	}

	//merge violations that only differ in a name or message part that is equal to the cluster name
	namePattern = strings.Replace(namePattern, clusterName, "<cluster>", -1)
	messagePattern = strings.Replace(messagePattern, clusterName, "<cluster>", -1)

	//same thing, but also allow a match only on region name
	regionName := regionNameInClusterNameRx.FindStringSubmatch(clusterName)[1]
	namePattern = strings.Replace(namePattern, regionName, "<region>", -1)
	messagePattern = strings.Replace(messagePattern, regionName, "<region>", -1)

	return ViolationGroup{
		Kind:              computedKind,
		NamePattern:       namePattern,
		Namespace:         namespacePattern,
		SupportGroupLabel: supportGroupLabel,
		ServiceLabel:      serviceLabel,
		Message:           messagePattern,
		Instances: []ViolationInstance{{
			ClusterName: clusterName,
			Name:        report.Name,
		}},
	}
}

// CanMergeWith checks if both ViolationGroups are semantically identical and
// can be merged.
func (vg ViolationGroup) CanMergeWith(other ViolationGroup) bool {
	return vg.Kind == other.Kind && vg.Namespace == other.Namespace &&
		vg.SupportGroupLabel == other.SupportGroupLabel && vg.ServiceLabel == other.ServiceLabel &&
		vg.NamePattern == other.NamePattern && vg.Message == other.Message
}

// GroupViolationsInto processes the violations in this report into
// ViolationGroups, sorted by template kind.
func (r Report) GroupViolationsInto(apiData APIData, clusterName string, showAll bool) {
	for _, rt := range r.Templates {
		for _, rc := range rt.Configs {
			if !showAll && rc.Labels["on-prod-ui"] != "true" {
				continue
			}

			//collect source backrefs from this config
			if apiData.KindInfos[rt.Kind] == nil {
				apiData.KindInfos[rt.Kind] = &KindInfo{}
			}
			if sourceStr := rc.Annotations["template-source"]; sourceStr != "" {
				apiData.KindInfos[rt.Kind].addTemplateSource(sourceStr)
			}
			if sourceStr := rc.Annotations["constraint-source"]; sourceStr != "" {
				apiData.KindInfos[rt.Kind].addConstraintSource(sourceStr)
			}

		VIOLATION:
			for _, rv := range rc.Violations {
				//start with a fresh violation group for this violation...
				vgNew := NewViolationGroup(rv, clusterName)

				//...but prefer to merge it with an existing group
				for _, vgOld := range apiData.ViolationGroups[rt.Kind] {
					if vgOld.CanMergeWith(vgNew) {
						vgOld.Instances = append(vgOld.Instances, vgNew.Instances...)
						continue VIOLATION
					}
				}

				//otherwise it gets inserted on its own
				apiData.ViolationGroups[rt.Kind] = append(apiData.ViolationGroups[rt.Kind], &vgNew)
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
		if lhs.NamePattern != rhs.NamePattern {
			return lhs.NamePattern < rhs.NamePattern
		}
		if lhs.Kind != rhs.Kind {
			return lhs.Kind < rhs.Kind
		}
		if lhs.Namespace != rhs.Namespace {
			return lhs.Namespace < rhs.Namespace
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
