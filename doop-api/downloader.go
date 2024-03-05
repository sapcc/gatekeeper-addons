/*******************************************************************************
*
* Copyright 2021-2023 SAP SE
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
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/majewsky/schwift"
	"github.com/sapcc/go-bits/logg"

	"github.com/sapcc/gatekeeper-addons/internal/doop"
)

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

// GetReports returns all most recent doop-agent reports in their yet unparsed form.
func (d *Downloader) GetReports() (map[string]doop.Report, error) {
	objInfos, err := d.container.Objects().CollectDetailed()
	if err != nil {
		return nil, fmt.Errorf("cannot list reports in Swift: %w", err)
	}

	d.mutex.Lock()
	defer d.mutex.Unlock()

	//NOTE: `objInfo` is the latest information from Swift about a report object.
	//`objState` is what this process knows about a report object.
	result := make(map[string]doop.Report, len(objInfos))
	for _, objInfo := range objInfos {
		name := objInfo.Object.Name()
		objState := d.objects[name]

		if objState.NeedsUpdate(objInfo) {
			logg.Debug("pulling updated report for %s", name)
			if objState == nil {
				objState = &objectState{}
				d.objects[name] = objState
			}
			objState.SizeBytes = objInfo.SizeBytes
			objState.Etag = objInfo.Etag
			objState.LastModified = objInfo.LastModified
			payloadBytes, err := objInfo.Object.Download(nil).AsByteSlice()
			if err != nil {
				return nil, fmt.Errorf("cannot download report for %s from Swift: %w", name, err)
			}
			var payload doop.Report
			err = json.Unmarshal(payloadBytes, &payload)
			if err != nil {
				return nil, fmt.Errorf("cannot decode report for %s: %w", name, err)
			}
			payload.SetClusterName(name)
			objState.Payload = payload
		}

		result[name] = objState.Payload
	}

	return result, nil
}

type objectState struct {
	SizeBytes    uint64
	Etag         string
	LastModified time.Time
	Payload      doop.Report
}

func (os *objectState) NeedsUpdate(oi schwift.ObjectInfo) bool {
	//if we don't have any state locally yet, we definitely need to update
	if os == nil {
		return true
	}
	return os.SizeBytes != oi.SizeBytes || os.Etag != oi.Etag || os.LastModified != oi.LastModified
}
