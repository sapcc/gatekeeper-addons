/*******************************************************************************
*
* Copyright 2023 SAP SE
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
	"io"
	"net/http"

	"github.com/NYTimes/gziphandler"
	"github.com/gorilla/mux"
	"github.com/sapcc/go-bits/httpapi"
	"github.com/sapcc/go-bits/logg"
	"github.com/sapcc/go-bits/respondwith"
)

// API is an httpapi.API implementation.
type API struct {
	Downloader *Downloader
}

// AddTo implements the httpapi.API interface.
func (a API) AddTo(r *mux.Router) {
	r.Methods("GET").Path("/v2/violations").Handler(gziphandler.GzipHandler(http.HandlerFunc(a.handleGetViolations)))
}

// The Gzip middleware will use the first few writes to decide whether to use compression or not
// because it stores the first write in its internal buffer, we must ensure that it doesn't grow too much.
type chunkingWriter struct {
	inner io.Writer
}

func (w chunkingWriter) Write(buf []byte) (int, error) {
	chunkSize := 1048576 // 1 MiB
	written := 0

	for len(buf) > chunkSize {
		n, err := w.inner.Write(buf[0:chunkSize])
		written += n
		buf = buf[chunkSize:]
		if err != nil {
			return written, err
		}
	}

	n, err := w.inner.Write(buf)
	return written + n, err
}

func (a API) handleGetViolations(w http.ResponseWriter, r *http.Request) {
	httpapi.IdentifyEndpoint(r, "/v2/violations")

	reports, err := a.Downloader.GetReports()
	if respondwith.ErrorText(w, err) {
		return
	}
	result := AggregateReports(reports, BuildFilterSet(r.URL.Query()))
	result.Sort()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	encoder := json.NewEncoder(chunkingWriter{inner: w})
	err = encoder.Encode(&result)
	if err != nil {
		logg.Error("while encoding violations json: %w", err)
		return
	}
}
