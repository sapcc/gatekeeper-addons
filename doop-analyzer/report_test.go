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
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/sapcc/go-bits/assert"
	"github.com/sapcc/go-bits/logg"
	"gopkg.in/yaml.v2"

	"github.com/sapcc/gatekeeper-addons/internal/util"
)

func TestGatherReport(t *testing.T) {
	// This test runs the dummy Kubernetes objects in `fixtures/gatekeeper/*.yaml`
	// through GatherReport and matches the result against `fixtures/report-before-processing.json`.

	cfg := Configuration{
		ClusterIdentity: map[string]string{"ci_key1": "ci_value1", "ci_key2": "ci_value2"},
	}
	report, err := GatherReport(context.Background(), cfg, mockClientSet{})
	if err != nil {
		t.Fatal(err.Error())
	}

	reportBuf, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err.Error())
	}
	assert.JSONFixtureFile("fixtures/report-before-processing.json").AssertResponseBody(t, "GatherReport", reportBuf)
}

type mockClientSet struct{}

func (mockClientSet) ListConstraintTemplates(ctx context.Context) ([]ConstraintTemplate, error) {
	return readItemListFromYAML[ConstraintTemplate]("fixtures/gatekeeper/constrainttemplates.yaml")
}

func (mockClientSet) ListConstraints(ctx context.Context, tmpl ConstraintTemplate) (result []Constraint, e error) {
	path := fmt.Sprintf("fixtures/gatekeeper/%s.yaml", tmpl.Metadata.Name)
	return readItemListFromYAML[Constraint](path)
}

func readItemListFromYAML[T any](path string) ([]T, error) {
	// Because the datatypes Constraint and ConstraintTemplate only have
	// annotations for JSON decoding, we need to take a slight detour
	// from YAML -> map[any]any -> map[string]any -> JSON -> []T.

	buf, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var data any
	err = yaml.Unmarshal(buf, &data)
	if err != nil {
		return nil, err
	}
	items, err := util.NormalizeRecursively("data", data)
	if err != nil {
		return nil, err
	}
	buf, err = json.Marshal(items)
	if err != nil {
		return nil, err
	}
	logg.Info(string(buf))
	var result struct {
		Items []T `json:"items"`
	}
	err = json.Unmarshal(buf, &result)
	return result.Items, err
}
