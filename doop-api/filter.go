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
	"strings"
)

// FilterSet describes which clusters/templates/constraints/violations to filter out when aggregating reports.
type FilterSet struct {
	clusterIdentity map[string]filter
	templateKind    filter
	constraintName  filter
	objectIdentity  map[string]filter
}

// BuildFilterSet collects filter settings from the given URL query.
func BuildFilterSet(query url.Values) FilterSet {
	return FilterSet{
		clusterIdentity: buildMapFilter(query, "cluster_identity."),
		templateKind:    filter(query["template_kind"]),
		constraintName:  filter(query["constraint_name"]),
		objectIdentity:  buildMapFilter(query, "object_identity."),
	}
}

func buildMapFilter(query url.Values, prefix string) map[string]filter {
	result := make(map[string]filter)
	for queryKey, values := range query {
		if strings.HasPrefix(queryKey, prefix) {
			key := strings.TrimPrefix(queryKey, prefix)
			result[key] = filter(values)
		}
	}
	return result
}

// MatchClusterIdentity checks whether a cluster with the given identity shall be included in the result.
func (fs FilterSet) MatchClusterIdentity(identity map[string]string) bool {
	for key, filter := range fs.clusterIdentity {
		if !filter.match(identity[key]) {
			return false
		}
	}
	return true
}

// MatchObjectIdentity checks whether a violation with the given object identity shall be included in the result.
func (fs FilterSet) MatchObjectIdentity(identity map[string]string) bool {
	for key, filter := range fs.objectIdentity {
		if !filter.match(identity[key]) {
			return false
		}
	}
	return true
}

// MatchTemplateKind checks whether a constraint template with the given kind shall be included in the result.
func (fs FilterSet) MatchTemplateKind(kind string) bool {
	return fs.templateKind.match(kind)
}

// MatchConstraintName checks whether a constraint with the given name shall be included in the result.
func (fs FilterSet) MatchConstraintName(name string) bool {
	return fs.constraintName.match(name)
}

// A list of allowed values for a certain field. If the list is empty, all values are allowed.
type filter []string

func (f filter) match(givenValue string) bool {
	if len(f) == 0 {
		return true
	}
	for _, allowedValue := range f {
		if allowedValue == givenValue {
			return true
		}
	}
	return false
}
