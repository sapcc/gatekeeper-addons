// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company
// SPDX-License-Identifier: Apache-2.0

package helmv3

// ReleaseContents is the data structure that helm-manifest-parser returns as
// JSON, and which helm-manifest-generator takes in as YAML.
type ReleaseContents struct {
	Items     []any             `json:"items" yaml:"items"`
	Values    any               `json:"values" yaml:"values"`
	OwnerInfo map[string]string `json:"owner_info" yaml:"owner-info"`

	// This section is only used by helm-manifest-generator.
	Metadata struct {
		Name      string `yaml:"name"`
		Namespace string `yaml:"namespace"` // default = release name
		Status    string `yaml:"status"`
		Version   uint   `yaml:"version"` // default = 1
	} `json:"-" yaml:"metadata"`
}

// The data structure inside the `data.release` of a Helm 3 release Secret.
type releasePayload struct {
	Name      string              `json:"name"`
	Info      releasePayloadInfo  `json:"info"`
	Chart     releasePayloadChart `json:"chart"`
	Values    any                 `json:"values"`
	Config    any                 `json:"config"`
	Manifest  string              `json:"manifest"`
	Version   uint                `json:"version"`
	Namespace string              `json:"namespace"`
}

type releasePayloadInfo struct {
	FirstDeployed string `json:"first_deployed"`
	LastDeployed  string `json:"last_deployed"`
	Deleted       string `json:"deleted"`
	Description   string `json:"description"`
	Status        string `json:"status"`
}

type releasePayloadChart struct {
	Metadata  releasePayloadChartMetadata   `json:"metadata"`
	Lock      releasePayloadChartLock       `json:"lock"`
	Templates []releasePayloadChartTemplate `json:"templates"`
}

type releasePayloadChartMetadata struct {
	Name         string                                  `json:"name"`
	Version      string                                  `json:"version"`
	Description  string                                  `json:"description"`
	APIVersion   string                                  `json:"apiVersion"`
	Dependencies []releasePayloadChartMetadataDependency `json:"dependencies"`
}

type releasePayloadChartMetadataDependency struct {
	Name       string `json:"name"`
	Version    string `json:"version"`
	Repository string `json:"repository"`
	Enabled    bool   `json:"enabled"`
}

type releasePayloadChartLock struct {
	Generated    string                              `json:"generated"`
	Digest       string                              `json:"digest"`
	Dependencies []releasePayloadChartLockDependency `json:"dependencies"`
}

type releasePayloadChartLockDependency struct {
	Name       string `json:"name"`
	Version    string `json:"version"`
	Repository string `json:"repository"`
}

type releasePayloadChartTemplate struct {
	Name string `json:"name"`
	Data string `json:"data"`
}
