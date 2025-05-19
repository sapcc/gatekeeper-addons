// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company
// SPDX-License-Identifier: Apache-2.0

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
