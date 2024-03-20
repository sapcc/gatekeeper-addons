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
	"net/http"
	"sync"
	"time"
)

type vulnCheckResult struct {
	Headers   http.Header
	CheckedAt time.Time
}

var (
	cache      = make(map[string]vulnCheckResult)
	cacheMutex sync.Mutex
)

func checkHeaderCache(imageRefStr string) (http.Header, bool) {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()

	result, ok := cache[imageRefStr]
	if !ok {
		return nil, false
	}

	if result.CheckedAt.Add(30 * time.Minute).Before(time.Now()) {
		// clear old cache entry
		delete(cache, imageRefStr)
		return nil, false
	}

	return result.Headers, ok
}

func fillHeaderCache(imageRefStr string, hdr http.Header) {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()
	cache[imageRefStr] = vulnCheckResult{hdr, time.Now()}
}
