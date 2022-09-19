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

package helmv3

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"

	"github.com/mitchellh/mapstructure"
	"gopkg.in/yaml.v2"

	"github.com/sapcc/gatekeeper-addons/internal/util"
)

// ParseRelease parses the `data.release` field of a Helm 3 release Secret.
func ParseRelease(in []byte) (*ReleaseContents, error) {
	var err error

	in, err = base64.StdEncoding.DecodeString(string(in))
	if err != nil {
		return nil, fmt.Errorf("cannot decode Base64: %w", err)
	}

	in, err = base64.StdEncoding.DecodeString(string(in))
	if err != nil {
		return nil, fmt.Errorf("cannot decode Base64: %w", err)
	}

	in, err = gunzip(in)
	if err != nil {
		return nil, fmt.Errorf("cannot decompress GZip: %w", err)
	}

	var parsed releasePayload
	err = json.Unmarshal(in, &parsed)
	if err != nil {
		return nil, fmt.Errorf("cannot parse Protobuf: %w", err)
	}

	var result ReleaseContents
	result.Items, err = convertManifestToItemsList([]byte(parsed.Manifest))
	if err != nil {
		return nil, fmt.Errorf("in manifest %s.v%d: %w", parsed.Name, parsed.Version, err)
	}
	result.Values, err = util.NormalizeRecursively(".values", parsed.Config)
	if err != nil {
		return nil, fmt.Errorf("in manifest %s.v%d: %w", parsed.Name, parsed.Version, err)
	}
	result.OwnerInfo, err = extractOwnerInfo(parsed.Name, result.Items)
	if err != nil {
		return nil, fmt.Errorf("in manifest %s.v%d: %w", parsed.Name, parsed.Version, err)
	}

	result.Metadata.Name = parsed.Name
	result.Metadata.Namespace = parsed.Namespace
	result.Metadata.Status = parsed.Info.Status
	result.Metadata.Version = parsed.Version

	return &result, nil
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
		val, err = util.NormalizeRecursively(fmt.Sprintf(".items[%d]", idx), val)
		if err != nil {
			return nil, err
		}
		result = append(result, val)
	}
	return result, nil
}

func extractOwnerInfo(releaseName string, items []interface{}) (map[string]string, error) {
	configMapName := "owner-of-" + releaseName

	//try to find the owner-info ConfigMap among all the manifest items
	for _, item := range items {
		var obj struct {
			Kind     string `mapstructure:"kind"`
			Metadata struct {
				Name string `mapstructure:"name"`
			} `mapstructure:"metadata"`
			Data interface{} `mapstructure:"data"` //cannot use a specific type here because we don't know which Kind of object we have yet
		}
		err := mapstructure.Decode(item, &obj)
		if err != nil {
			return nil, fmt.Errorf("while looking for owner-info: %w", err)
		}

		if obj.Kind == "ConfigMap" && obj.Metadata.Name == configMapName {
			var result map[string]string
			err = mapstructure.Decode(obj.Data, &result)
			return result, err
		}
	}

	//no owner-info ConfigMap found
	return map[string]string{}, nil
}
