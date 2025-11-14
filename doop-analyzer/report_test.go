// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/sapcc/go-bits/assert"
)

func TestGatherReport(t *testing.T) {
	// This test runs the dummy Kubernetes objects in `fixtures/gatekeeper/*.json`
	// through GatherReport and matches the result against `fixtures/report-before-processing.json`.

	cfg := Configuration{
		ClusterIdentity: map[string]string{"ci_key1": "ci_value1", "ci_key2": "ci_value2"},
	}
	report, err := GatherReport(t.Context(), cfg, mockClientSet{})
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
	return readItemListFromJSON[ConstraintTemplate]("fixtures/gatekeeper/constrainttemplates.json")
}

func (mockClientSet) ListConstraints(ctx context.Context, tmpl ConstraintTemplate) (result []Constraint, e error) {
	path := fmt.Sprintf("fixtures/gatekeeper/%s.json", tmpl.Metadata.Name)
	return readItemListFromJSON[Constraint](path)
}

func readItemListFromJSON[T any](path string) ([]T, error) {
	buf, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var result struct {
		Items []T `json:"items"`
	}
	err = json.Unmarshal(buf, &result)
	return result.Items, err
}
