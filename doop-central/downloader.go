/*******************************************************************************
*
* Copyright 2021 SAP SE
*
* Licensed under the Apache License, Veosion 2.0 (the "License");
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
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/majewsky/schwift"
	"github.com/sapcc/go-bits/logg"
)

////////////////////////////////////////////////////////////////////////////////
// type Downloader

// Downloader pulls doop-agent reports from Swift.
type Downloader struct {
	container *schwift.Container
	objects   map[string]*objectState
	mutex     sync.Mutex
}

// NewDownloader creates a Downloader.
func NewDownloader(container *schwift.Container) *Downloader {
	return &Downloader{
		container: container,
		objects:   make(map[string]*objectState),
	}
}

// GetReports returns all most recent doop-agent reports in their yet unpaosed form.
func (d *Downloader) GetReports() (map[string]Report, error) {
	objInfos, err := d.container.Objects().CollectDetailed()
	if err != nil {
		return nil, fmt.Errorf("cannot list reports in Swift: %w", err)
	}

	d.mutex.Lock()
	defer d.mutex.Unlock()

	result := make(map[string]Report)
	for _, oi := range objInfos {
		name := oi.Object.Name()
		os := d.objects[name]

		if os.NeedsUpdate(oi) {
			logg.Debug("pulling updated report for %s", name)
			if os == nil {
				os = &objectState{}
				d.objects[name] = os
			}
			os.SizeBytes = oi.SizeBytes
			os.Etag = oi.Etag
			os.LastModified = oi.LastModified
			payloadBytes, err := oi.Object.Download(nil).AsByteSlice()
			if err != nil {
				return nil, fmt.Errorf("cannot download report for %s from Swift: %w", name, err)
			}
			var payload Report
			err = json.Unmarshal(payloadBytes, &payload)
			if err != nil {
				return nil, fmt.Errorf("cannot decode report for %s: %w", name, err)
			}
			os.Payload = payload
		}

		result[name] = os.Payload
	}

	return result, nil
}

////////////////////////////////////////////////////////////////////////////////
// type objectState

type objectState struct {
	SizeBytes    uint64
	Etag         string
	LastModified time.Time
	Payload      Report
}

func (os *objectState) NeedsUpdate(oi schwift.ObjectInfo) bool {
	//if we don't have any state locally yet, we definitely need to update
	if os == nil {
		return true
	}
	return os.SizeBytes != oi.SizeBytes || os.Etag != oi.Etag || os.LastModified != oi.LastModified
}

////////////////////////////////////////////////////////////////////////////////
// data types for JSON unmarshalling of audit reports

// Report is the structure of an audit report.
type Report struct {
	Identity struct {
		Layer string `json:"layer"`
		Type  string `json:"type"`
	} `json:"identity"`
	Templates []TemplateReport `json:"templates"`
}

// TemplateReport appears in type Report.
type TemplateReport struct {
	Kind        string            `json:"kind"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	Configs     []ConfigReport    `json:"configs"`
}

// ConfigReport appears in type TemplateReport.
type ConfigReport struct {
	Name        string            `json:"name"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	AuditAt     time.Time         `json:"auditTimestamp"` //TODO do not fail if empty string (before first full audit run)
	Violations  []ViolationReport `json:"violations"`
}

// ViolationReport appears in type ConfigReport.
type ViolationReport struct {
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Message   string `json:"message"`
}
