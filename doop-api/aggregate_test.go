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
	"encoding/json"
	"net/url"
	"os"
	"testing"

	"github.com/sapcc/go-bits/assert"

	"github.com/sapcc/gatekeeper-addons/internal/doop"
)

func TestAggregateOfOneCluster(t *testing.T) {
	inputSet := map[string]doop.Report{
		"cluster1": mustParseJSON[doop.Report](t, "fixtures/input-cluster1.json"),
	}

	//test that aggregating results from only one cluster barely changes the input if no filter is applied
	expected := mustParseJSON[doop.AggregatedReport](t, "fixtures/output-cluster1-only.json")
	actual := AggregateReports(inputSet, BuildFilterSet(url.Values{}))
	assert.DeepEqual(t, "AggregateReports", actual, expected)

	//test a filter that does not change anything because it exactly matches what is in the report
	filterStr := "cluster_identity.number=one&template_kind=GkFirstTemplate&constraint_name=firstconstraint&object_identity.type=production"
	actual = AggregateReports(inputSet, BuildFilterSet(query(filterStr)))
	assert.DeepEqual(t, "AggregateReports", actual, expected)

	//test a filter that removes all clusters
	filterStr = "cluster_identity.number=two"
	actual = AggregateReports(inputSet, BuildFilterSet(query(filterStr)))
	assert.DeepEqual(t, "AggregateReports", actual, doop.AggregatedReport{
		ClusterIdentities: map[string]map[string]string{},
		Templates:         nil,
	})

	//test several filters that remove all violations because they mismatch on each other possible level
	//(removing violations also removes all effectively empty objects above it)
	negativeFilters := []string{
		"template_kind=GkSecondTemplate",
		"constraint_name=secondconstraint",
		"object_identity.type=qa",
	}
	for _, filterStr := range negativeFilters {
		actual = AggregateReports(inputSet, BuildFilterSet(query(filterStr)))
		assert.DeepEqual(t, "AggregateReports", actual, doop.AggregatedReport{
			ClusterIdentities: map[string]map[string]string{
				"cluster1": {"number": "one"},
			},
			Templates: nil,
		})
	}
}

func TestAggregateOfTwoClusters(t *testing.T) {
	//Each of these cluster reports has exactly one violation.
	inputSet := map[string]doop.Report{
		"cluster1": mustParseJSON[doop.Report](t, "fixtures/input-cluster1.json"),
		//this one can merge with cluster 1 on same violation group
		"cluster2": mustParseJSON[doop.Report](t, "fixtures/input-cluster2.json"),
		//this one can merge with cluster 1 on different violation group, but same constraint
		"cluster3": mustParseJSON[doop.Report](t, "fixtures/input-cluster3.json"),
		//this one can merge with cluster 1 on different constraint, but same template
		"cluster4": mustParseJSON[doop.Report](t, "fixtures/input-cluster4.json"),
	}

	//test merging of structures on all levels of the report
	expected := mustParseJSON[doop.AggregatedReport](t, "fixtures/output-both-clusters.json")
	actual := AggregateReports(inputSet, BuildFilterSet(url.Values{}))
	assert.DeepEqual(t, "AggregateReports", actual, expected)
}

func query(input string) url.Values {
	result, err := url.ParseQuery(input)
	if err != nil {
		panic(err.Error())
	}
	return result
}

func mustParseJSON[T any](t *testing.T, path string) (result T) {
	t.Helper()
	buf, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err.Error())
	}
	err = json.Unmarshal(buf, &result)
	if err != nil {
		t.Fatal(err.Error())
	}
	return result
}
