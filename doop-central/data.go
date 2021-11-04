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
	"html/template"
	"sort"
)

type PreprocessedData struct {
	ShowAll          bool
	AllClusters      []string
	AllClusterLayers []string
	AllClusterTypes  []string
	ClusterInfos     map[string]ClusterInfo
	AllTemplateKinds []string
	ViolationGroups  map[string][]*ViolationGroup
	Docstrings       map[string]template.HTML
}

func (downloader *Downloader) retrieveData(showAll bool) (PreprocessedData, error) {
	reports, err := downloader.GetReports()
	if err != nil {
		return PreprocessedData{}, err
	}

	data := PreprocessedData{
		ClusterInfos:    make(map[string]ClusterInfo),
		ViolationGroups: make(map[string][]*ViolationGroup),
		ShowAll:         showAll,
	}

	for clusterName, report := range reports {
		data.AllClusters = append(data.AllClusters, clusterName)
		data.AllClusterLayers = append(data.AllClusterLayers, report.Identity.Layer)
		data.AllClusterTypes = append(data.AllClusterTypes, report.Identity.Type)
		data.ClusterInfos[clusterName] = report.ToClusterInfo()
		report.GroupViolationsInto(data.ViolationGroups, clusterName, data.ShowAll)
	}

	sort.Strings(data.AllClusters)
	data.AllClusterLayers = sortAndDedupStrings(data.AllClusterLayers)
	data.AllClusterTypes = sortAndDedupStrings(data.AllClusterTypes)
	for kind, violationGroups := range data.ViolationGroups {
		data.AllTemplateKinds = append(data.AllTemplateKinds, kind)
		sortViolationGroups(violationGroups)
	}
	sort.Strings(data.AllTemplateKinds)

	return data, nil
}
