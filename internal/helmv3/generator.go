/*******************************************************************************
*
* Copyright 2022 SAP SE
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
	"strconv"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"github.com/mitchellh/mapstructure"
	"gopkg.in/yaml.v2"

	"github.com/sapcc/gatekeeper-addons/internal/util"
)

const (
	mockChartVersion = "0.0.1"
	mockLockDigest   = "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
)

// GenerateMockRelease is kind of the reverse of ParseRelease(). It generates
// a synthetic release object that is a good-enough approximation of an actual
// Helm 3 release Secret to be used for unit tests.
func (c ReleaseContents) GenerateMockRelease() (interface{}, error) {
	//check required values
	if c.Metadata.Name == "" {
		return nil, fmt.Errorf("missing required field: metadata.name")
	}
	if c.Metadata.Status == "" {
		return nil, fmt.Errorf("missing required field: metadata.status")
	}

	//apply default values
	if c.Metadata.Namespace == "" {
		c.Metadata.Namespace = c.Metadata.Name
	}
	if c.Metadata.Version == 0 {
		c.Metadata.Version = 1
	}
	if c.Values == nil {
		c.Values = map[string]interface{}{}
	}

	//generate some plausible mock values
	mockUpgradeTime := time.Now().UTC()
	mockInstallTime := mockUpgradeTime
	if c.Metadata.Version > 1 {
		mockInstallTime = mockUpgradeTime.Add(-24 * time.Hour).UTC()
	}
	mockStatusDesc := fmt.Sprintf("Moving into status %s", c.Metadata.Status)
	mockChartDesc := fmt.Sprintf("Helm chart for %s", c.Metadata.Name)
	mockSecretResourceVersion := strconv.FormatInt(23*mockUpgradeTime.Unix(), 10)
	mockSecretUUID, err := uuid.NewV4()
	if err != nil {
		return nil, fmt.Errorf("while generating UUID for Secret: %w", err)
	}

	//generate payload
	normalizedValues, err := util.NormalizeRecursively(".Values", c.Values)
	if err != nil {
		return nil, err
	}
	payload := releasePayload{
		Name: c.Metadata.Name,
		Info: releasePayloadInfo{
			FirstDeployed: mockInstallTime.Format(time.RFC3339Nano),
			LastDeployed:  mockUpgradeTime.Format(time.RFC3339Nano),
			Deleted:       "",
			Description:   mockStatusDesc,
			Status:        c.Metadata.Status,
		},
		Chart: releasePayloadChart{
			Metadata: releasePayloadChartMetadata{
				Name:         c.Metadata.Name,
				Version:      mockChartVersion,
				Description:  mockChartDesc,
				APIVersion:   "v2",
				Dependencies: []releasePayloadChartMetadataDependency{}, //filled below
			},
			Lock: releasePayloadChartLock{
				Generated:    mockInstallTime.Format(time.RFC3339Nano),
				Digest:       mockLockDigest,
				Dependencies: []releasePayloadChartLockDependency{}, //filled below
			},
			Templates: []releasePayloadChartTemplate{}, //filled below
		},
		Config:    normalizedValues,
		Values:    map[string]interface{}{}, //TODO if this is required in the future, we need to figure out how it is structured
		Manifest:  "",                       //filled below
		Version:   c.Metadata.Version,
		Namespace: c.Metadata.Namespace,
	}

	//add manifest objects to payload
	addManifestObject := func(buf []byte, kind, name string) {
		mockFileName := strings.ToLower(fmt.Sprintf("templates/%s-%s.yaml", kind, name))
		payload.Chart.Templates = append(payload.Chart.Templates, releasePayloadChartTemplate{
			Name: mockFileName,
			Data: base64.StdEncoding.EncodeToString(buf),
		})
		payload.Manifest += fmt.Sprintf("---\n# Source: %s/%s\n%s", c.Metadata.Name, mockFileName, string(buf))
	}
	manifestHasOwnerInfo := false
	for idx, item := range c.Items {
		var info struct {
			Kind     string `mapstructure:"kind"`
			Metadata struct {
				Name string `mapstructure:"name"`
			} `mapstructure:"metadata"`
		}
		err := mapstructure.Decode(item, &info)
		if err != nil {
			return nil, fmt.Errorf("cannot collect Kind and Name from .items[%d]: %w", idx, err)
		}
		buf, err := yaml.Marshal(item)
		if err != nil {
			return nil, fmt.Errorf("cannot render YAML for .items[%d]: %w", idx, err)
		}
		addManifestObject(buf, info.Kind, info.Metadata.Name)
		if info.Kind == "ConfigMap" && info.Metadata.Name == "owner-of-"+c.Metadata.Name {
			manifestHasOwnerInfo = true
		}
	}

	//add owner-info to payload
	if c.OwnerInfo != nil {
		//when starting from a YAML-encoded ReleaseContents, `c.Items` will not
		//contain the owner-info ConfigMap and we need to add it now; if we started
		//from an actual Helm release manifest, `c.Items` will have the owner-info
		//ConfigMap (among other things) and we should not duplicate it to avoid confusion
		if !manifestHasOwnerInfo {
			name := "owner-of-" + c.Metadata.Name
			item := map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "ConfigMap",
				"metadata": map[string]interface{}{
					"name": name,
				},
				"data": c.OwnerInfo,
			}
			buf, err := yaml.Marshal(item)
			if err != nil {
				return nil, fmt.Errorf("cannot render YAML for owner-info ConfigMap: %w", err)
			}
			addManifestObject(buf, "ConfigMap", name)
		}

		//add chart dependency to owner-info
		payload.Chart.Metadata.Dependencies = append(payload.Chart.Metadata.Dependencies, releasePayloadChartMetadataDependency{
			Name:       "owner-info",
			Version:    "0.2.0", //TODO allow to configure
			Repository: "https://charts.eu-de-2.cloud.sap",
			Enabled:    true,
		})

		//add lock dependency to owner-info
		payload.Chart.Lock.Dependencies = append(payload.Chart.Lock.Dependencies, releasePayloadChartLockDependency{
			Name:       "owner-info",
			Version:    "0.2.0", //TODO allow to configure
			Repository: "https://charts.eu-de-2.cloud.sap",
		})
	}

	//pack payload
	buf, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("cannot serialize payload: %w", err)
	}
	buf, err = gzipCompress(buf)
	if err != nil {
		return nil, fmt.Errorf("cannot compress payload: %w", err)
	}
	packedPayload := base64.StdEncoding.EncodeToString([]byte(base64.StdEncoding.EncodeToString(buf)))

	//wrap payload into a k8s Secret object
	return map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "Secret",
		"metadata": map[string]interface{}{
			"creationTimestamp": mockUpgradeTime.Format(time.RFC3339),
			"labels": map[string]string{
				"modifiedAt": strconv.FormatInt(mockUpgradeTime.Unix(), 10),
				"name":       c.Metadata.Name,
				"owner":      "helm",
				"status":     c.Metadata.Status,
				"version":    strconv.FormatUint(uint64(c.Metadata.Version), 10),
			},
			"name":            fmt.Sprintf("sh.helm.release.v1.%s.v%d", c.Metadata.Name, c.Metadata.Version),
			"namespace":       c.Metadata.Namespace,
			"resourceVersion": mockSecretResourceVersion,
			"uuid":            mockSecretUUID.String(),
		},
		"type": "helm.sh/release.v1",
		"data": map[string]string{"release": packedPayload},
	}, nil
}

func gzipCompress(in []byte) ([]byte, error) {
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	_, err := w.Write(in)
	if err != nil {
		return nil, err
	}
	err = w.Close()
	return buf.Bytes(), err
}
