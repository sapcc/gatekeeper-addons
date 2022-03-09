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
		{"fixtures/helm-v3-manifest.dat", "fixtures/helm-v3-parsed.json", ParseHelm3Manifest},
	}

	for _, tc := range testCases {
		inBytes, err := os.ReadFile(tc.InputPath)
		must(t, err)
		outStr, err := tc.Parser(bytes.TrimSpace(inBytes))
		must(t, err)

		//in order to diff(1) with `tc.OutputPath`, we need to apply the same indentation
		var outBuf bytes.Buffer
		must(t, json.Indent(&outBuf, []byte(outStr), "", "  "))
		outBuf.WriteString("\n")
		must(t, os.WriteFile(tc.OutputPath+".actual", outBuf.Bytes(), 0666))

		cmd := exec.Command("diff", "-u", tc.OutputPath, tc.OutputPath+".actual")
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
