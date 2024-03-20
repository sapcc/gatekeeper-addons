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
