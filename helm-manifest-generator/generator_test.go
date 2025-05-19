// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company
// SPDX-License-Identifier: Apache-2.0

package main_test

import (
	"os"
	"os/exec"
	"testing"

	"gopkg.in/yaml.v2"

	"github.com/sapcc/gatekeeper-addons/internal/helmv3"
)

func TestRoundtrip(t *testing.T) {
	testCases := []struct {
		InputPath        string
		RoundtrippedPath string
	}{
		{"fixtures/test1-input.yaml", "fixtures/test1-roundtripped.yaml"},
	}

	for _, tc := range testCases {
		inBytes, err := os.ReadFile(tc.InputPath)
		must(t, err)

		// generate mock release from input declaration
		var inContents helmv3.ReleaseContents
		must(t, yaml.Unmarshal(inBytes, &inContents))
		secretObj, err := inContents.GenerateMockRelease()
		must(t, err)

		// roundtrip back into a declaration
		roundtrippedContents, err := helmv3.ParseRelease([]byte(secretObj.(map[string]any)["data"].(map[string]string)["release"]))
		must(t, err)
		roundtrippedBytes, err := yaml.Marshal(roundtrippedContents)
		must(t, err)

		// diff against expectation
		must(t, os.WriteFile(tc.RoundtrippedPath+".actual", roundtrippedBytes, 0o666))
		cmd := exec.Command("diff", "-u", tc.RoundtrippedPath, tc.RoundtrippedPath+".actual") //nolint:gosec // command only executed in tests
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		must(t, cmd.Run())
	}
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err.Error())
	}
}
