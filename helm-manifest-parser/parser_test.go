// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"testing"
)

func TestParseManifests(t *testing.T) {
	testCases := []struct {
		InputPath  string
		OutputPath string
		Parser     func([]byte) (string, error)
	}{
		{"fixtures/helm-v3-manifest.dat", "fixtures/helm-v3-parsed.json", helm3parse},
		{"fixtures/helm-v3-with-ownerinfo-manifest.dat", "fixtures/helm-v3-with-ownerinfo-parsed.json", helm3parse},
	}

	for _, tc := range testCases {
		inBytes, err := os.ReadFile(tc.InputPath)
		mustT(t, err)
		outStr, err := tc.Parser(bytes.TrimSpace(inBytes))
		mustT(t, err)

		// in order to diff(1) with `tc.OutputPath`, we need to apply the same indentation
		var outBuf bytes.Buffer
		mustT(t, json.Indent(&outBuf, []byte(outStr), "", "  "))
		outBuf.WriteString("\n")
		mustT(t, os.WriteFile(tc.OutputPath+".actual", outBuf.Bytes(), 0o666))

		cmd := exec.Command("diff", "-u", tc.OutputPath, tc.OutputPath+".actual") //nolint:gosec // command only executed in tests
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		mustT(t, cmd.Run())
	}
}

func mustT(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err.Error())
	}
}
