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
	"io/ioutil"
	"strconv"

	"github.com/golang/protobuf/proto"
	"gopkg.in/yaml.v2"
	"k8s.io/helm/pkg/proto/hapi/release"
)

//ParseHelm2Manifest parses the `data.release` field of a Helm 2 release ConfigMap.
func ParseHelm2Manifest(in []byte) (string, error) {
	var err error

	in, err = base64.StdEncoding.DecodeString(string(in))
	if err != nil {
		return "", fmt.Errorf("cannot decode Base64: %w", err)
	}

	in, err = gunzip(in)
	if err != nil {
		return "", fmt.Errorf("cannot decompress GZip: %w", err)
	}

	var parsed release.Release
	err = proto.Unmarshal(in, &parsed)
	if err != nil {
		return "", fmt.Errorf("cannot parse Protobuf: %w", err)
	}

	out, err := convertManifestToJSON([]byte(parsed.Manifest))
	if err != nil {
		return "", fmt.Errorf("in manifest %s.v%d: %w", parsed.Name, parsed.Version, err)
	}
	return out, nil
}

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
		Name     string `json:"name"`
		Version  int    `json:"version"`
		Manifest string `json:"manifest"`
	}
	err = json.Unmarshal(in, &parsed)
	if err != nil {
		return "", fmt.Errorf("cannot parse Protobuf: %w", err)
	}

	out, err := convertManifestToJSON([]byte(parsed.Manifest))
	if err != nil {
		return "", fmt.Errorf("in manifest %s.v%d: %w", parsed.Name, parsed.Version, err)
	}
	return out, nil
}

func gunzip(in []byte) ([]byte, error) {
	r, err := gzip.NewReader(bytes.NewReader(in))
	if err != nil {
		return nil, err
	}
	return ioutil.ReadAll(r)
}

func convertManifestToJSON(in []byte) (string, error) {
	//yaml.Unmarshal into a generic map will create map[interface{}]interface{}
	//members which json.Marshal() cannot process. We need to convert these into
	//map[string]interface{} recursively before proceeding.
	normalizeKey := func(path string, k interface{}) (string, error) {
		switch k := k.(type) {
		case string:
			return k, nil
		case int:
			//has been observed in the wild in `.data` of a v1/Secret
			return strconv.Itoa(k), nil
		default:
			return "", fmt.Errorf("non-string key at %s: %T %v", path, k, k)
		}
	}

	var normalizeRecursively func(path string, in interface{}) (interface{}, error)
	normalizeRecursively = func(path string, in interface{}) (interface{}, error) {
		switch in := in.(type) {
		case map[interface{}]interface{}:
			out := make(map[string]interface{}, len(in))
			for k, v := range in {
				kn, err := normalizeKey(path, k)
				if err != nil {
					return nil, err
				}
				vn, err := normalizeRecursively(fmt.Sprintf("%s.%s", path, kn), v)
				if err != nil {
					return nil, err
				}
				out[kn] = vn
			}
			return out, nil
		case []interface{}:
			out := make([]interface{}, len(in))
			for idx, v := range in {
				vn, err := normalizeRecursively(fmt.Sprintf("%s[%d]", path, idx), v)
				if err != nil {
					return nil, err
				}
				out[idx] = vn
			}
			return out, nil
		default:
			return in, nil
		}
	}

	var result struct {
		Items []interface{} `json:"items"`
	}
	result.Items = []interface{}{} //ensure that empty list is rendered as [] rather than null

	dec := yaml.NewDecoder(bytes.NewReader(in))
	for idx := 0; ; idx++ {
		var val interface{}
		err := dec.Decode(&val)
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("cannot unmarshal YAML objects[%d]: %w", idx, err)
		}
		val, err = normalizeRecursively(fmt.Sprintf(".items[%d]", idx), val)
		if err != nil {
			return "", err
		}
		result.Items = append(result.Items, val)
	}

	out, err := json.Marshal(result)
	return string(out), err
}
