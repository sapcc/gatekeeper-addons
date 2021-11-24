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
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"

	"gopkg.in/yaml.v2"
)

//ParseHelm3Manifest parses the `data.release` field of a Helm 2 release ConfigMap.
func ParseHelm3Manifest(in []byte) (string, error) {
	var err error

	in, err = base64.StdEncoding.DecodeString(string(in))
	if err != nil {
		return "", fmt.Errorf("cannot decode Base64: %w", err)
	}

	in, err = base64.StdEncoding.DecodeString(string(in))
	if err != nil {
		return "", fmt.Errorf("cannot decode Base64: %w", err)
	}

	in, err = gunzip(in)
	if err != nil {
		return "", fmt.Errorf("cannot decompress GZip: %w", err)
	}

	var parsed struct {
		Name     string      `json:"name"`
		Version  int         `json:"version"`
		Manifest string      `json:"manifest"`
		Values   interface{} `json:"config"`
	}
	err = json.Unmarshal(in, &parsed)
	if err != nil {
		return "", fmt.Errorf("cannot parse Protobuf: %w", err)
	}

	var result struct {
		Items  []interface{} `json:"items"`
		Values interface{}   `json:"values"`
	}

	result.Items, err = convertManifestToItemsList([]byte(parsed.Manifest))
	if err != nil {
		return "", fmt.Errorf("in manifest %s.v%d: %w", parsed.Name, parsed.Version, err)
	}
	result.Values, err = NormalizeRecursively(".values", parsed.Values)
	if err != nil {
		return "", fmt.Errorf("in manifest %s.v%d: %w", parsed.Name, parsed.Version, err)
	}

	out, err := json.Marshal(result)
	return string(out), err
}

func gunzip(in []byte) ([]byte, error) {
	r, err := gzip.NewReader(bytes.NewReader(in))
	if err != nil {
		return nil, err
	}
	return io.ReadAll(r)
}

func convertManifestToItemsList(in []byte) ([]interface{}, error) {
	result := []interface{}{} //ensure that empty list is rendered as [] rather than null

	dec := yaml.NewDecoder(bytes.NewReader(in))
	for idx := 0; ; idx++ {
		var val interface{}
		err := dec.Decode(&val)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("cannot unmarshal YAML objects[%d]: %w", idx, err)
		}
		val, err = NormalizeRecursively(fmt.Sprintf(".items[%d]", idx), val)
		if err != nil {
			return nil, err
		}
		result = append(result, val)
	}
	return result, nil
}
