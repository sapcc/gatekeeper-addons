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
	"os"
	"testing"

	"github.com/sapcc/go-bits/assert"
	"github.com/sapcc/go-bits/regexpext"
)

func TestProcessReport(t *testing.T) {
	//This test runs the report generated at the end of TestGatherReport() through Report.Process()
	//and matches the result against `fixtures/report-after-processing.json`.

	cfg := Configuration{
		ProcessingRules: []Rule{
			// This rule makes Helm 3 secrets more readable: For example,
			//
			//   kind = "Secret"
			//   name = "sh.helm.release.v1.foobar.v42"
			//
			// becomes:
			//
			//   kind = "Helm 3 release"
			//   name = "foobar.v42"
			//
			// This usecase is why the ProcessingRules phase was added in the first place, and also why
			// Rule.Replace.Target supports multiple targets.
			{
				Match: map[string]regexpext.BoundedRegexp{"kind": "Secret"},
				Replace: ReplaceRule{
					Source:  "name",
					Pattern: `sh\.helm\.release\.v1\.(.*\.v\d+)`,
					Target:  map[string]string{"kind": "Helm 3 release", "name": "$1"},
				},
			},
		},
		MergingRules: []Rule{
			// This rule is a companion for the processing rule above. In real-world
			// scenarios, we want to be able to merge violations for the same Helm
			// release across different clusters even if the version number diverges.
			{
				Match: map[string]regexpext.BoundedRegexp{"kind": "Helm 3 release"},
				Replace: ReplaceRule{
					Source:  "name",
					Pattern: `(.*)\.v\d+`,
					Target:  map[string]string{"name": "$1.<variable>"},
				},
			},
			// Similarly, we want to be able to merge violations across pods
			// belonging to the same DaemonSet or ReplicaSet. In a real-world
			// scenario, there will be additional rules to account for Deployments,
			// StatefulSets, etc. as well.
			{
				Match: map[string]regexpext.BoundedRegexp{"kind": "Pod"},
				Replace: ReplaceRule{
					Source:  "name",
					Pattern: `(.*)-[a-z0-9]{5}`,
					Target:  map[string]string{"name": "$1-<variable>"},
				},
			},
		},
	}

	buf, err := os.ReadFile("fixtures/report-before-processing.json")
	if err != nil {
		t.Fatal(err.Error())
	}
	var report Report
	err = json.Unmarshal(buf, &report)
	if err != nil {
		t.Fatal(err.Error())
	}

	report.Process(cfg)

	reportBuf, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err.Error())
	}
	assert.JSONFixtureFile("fixtures/report-after-processing.json").AssertResponseBody(t, "Report.Process", reportBuf)
}
