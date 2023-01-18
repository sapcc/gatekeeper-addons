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

// APIData is the data type that is returned by the GET /json and GET /json/all endpoints.
type APIData struct {
	ClusterInfos    map[string]ClusterInfo       `json:"clusters"`
	KindInfos       map[string]*KindInfo         `json:"kinds"`
	ViolationGroups map[string][]*ViolationGroup `json:"violation_groups"`
}

type PreprocessedData struct {
	ShowAll          bool
	AllClusters      []string
	AllClusterLayers []string
	AllClusterTypes  []string
	AllTemplateKinds []string
	AllSupportGroups []string
	AllServiceLabels []string
	APIData          APIData
	Docstrings       map[string]template.HTML
}

func (d *Downloader) retrieveData(showAll bool) (PreprocessedData, error) {
	reports, err := d.GetReports()
	if err != nil {
		return PreprocessedData{}, err
	}

	data := PreprocessedData{
		APIData: APIData{
			ClusterInfos:    make(map[string]ClusterInfo),
			KindInfos:       make(map[string]*KindInfo),
			ViolationGroups: make(map[string][]*ViolationGroup),
		},
		ShowAll: showAll,
	}

	for clusterName, report := range reports {
		data.AllClusters = append(data.AllClusters, clusterName)
		data.AllClusterLayers = append(data.AllClusterLayers, report.Identity.Layer)
		data.AllClusterTypes = append(data.AllClusterTypes, report.Identity.Type)
		data.APIData.ClusterInfos[clusterName] = report.ToClusterInfo()
		report.GroupViolationsInto(data.APIData, clusterName, data.ShowAll)
	}

	sort.Strings(data.AllClusters)
	data.AllClusterLayers = sortAndDedupStrings(data.AllClusterLayers)
	data.AllClusterTypes = sortAndDedupStrings(data.AllClusterTypes)
	for kind, violationGroups := range data.APIData.ViolationGroups {
		data.AllTemplateKinds = append(data.AllTemplateKinds, kind)
		for _, vg := range violationGroups {
			data.AllSupportGroups = append(data.AllSupportGroups, vg.SupportGroupLabel)
			data.AllServiceLabels = append(data.AllServiceLabels, vg.ServiceLabel)
		}
		sortViolationGroups(violationGroups)
	}
	sort.Strings(data.AllTemplateKinds)
	data.AllSupportGroups = sortAndDedupStrings(data.AllSupportGroups)
	data.AllServiceLabels = sortAndDedupStrings(data.AllServiceLabels)

	return data, nil
}
