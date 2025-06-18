// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"net/url"
	"slices"
	"strings"
)

// FilterSet describes which clusters/templates/constraints/violations to filter out when aggregating reports.
type FilterSet struct {
	clusterIdentity map[string]filter
	templateKind    filter
	constraintName  filter
	severity        filter
	objectIdentity  map[string]filter
}

// BuildFilterSet collects filter settings from the given URL query.
func BuildFilterSet(query url.Values) FilterSet {
	return FilterSet{
		clusterIdentity: buildMapFilter(query, "cluster_identity."),
		templateKind:    filter(query["template_kind"]),
		constraintName:  filter(query["constraint_name"]),
		severity:        filter(query["severity"]),
		objectIdentity:  buildMapFilter(query, "object_identity."),
	}
}

func buildMapFilter(query url.Values, prefix string) map[string]filter {
	result := make(map[string]filter)
	for queryKey, values := range query {
		if key, ok := strings.CutPrefix(queryKey, prefix); ok {
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

// MatchSeverity checks whether a constraint with the given severity shall be included in the result.
func (fs FilterSet) MatchSeverity(severity string) bool {
	return fs.severity.match(severity)
}

// A list of allowed values for a certain field. If the list is empty, all values are allowed.
type filter []string

func (f filter) match(givenValue string) bool {
	if len(f) == 0 {
		return true
	}
	return slices.Contains(f, givenValue)
}
