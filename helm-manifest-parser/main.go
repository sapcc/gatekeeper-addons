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
	"context"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/sapcc/go-bits/httpapi"
	"github.com/sapcc/go-bits/httpext"
	"github.com/sapcc/go-bits/logg"
	"github.com/sapcc/go-bits/osext"
)

func main() {
	if len(os.Args) != 2 {
		logg.Fatal("usage: %s <listen-address>", os.Args[0])
	}

	logAllRequests := osext.GetenvBool("LOG_ALL_REQUESTS")
	apis := []httpapi.API{
		api{},
		httpapi.HealthCheckAPI{},
	}
	if !logAllRequests {
		apis = append(apis, httpapi.WithoutLogging())
	}
	handler := httpapi.Compose(apis...)

	logg.Info("listening on " + os.Args[1])
	ctx := httpext.ContextWithSIGINT(context.Background(), 10*time.Second)
	err := httpext.ListenAndServeContext(ctx, os.Args[1], handler)
	if err != nil {
		logg.Fatal(err.Error())
	}
}

type api struct{}

func (api) AddTo(r *mux.Router) {
	r.Methods("POST").Path("/v3").HandlerFunc(handleAPI("/v3", ParseHelm3Manifest))
}

func handleAPI(path string, parser func([]byte) (string, error)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != path {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		if r.Method != "POST" {
			http.Error(w, "only POST requests are allowed", http.StatusMethodNotAllowed)
			return
		}

		//never read more than 4 MiB to avoid DoS
		in, err := io.ReadAll(io.LimitReader(r.Body, 4<<20))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		out, err := parser(in)
		if err != nil {
			logg.Info("HTTP 400: " + err.Error())
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(out)) //nolint:errcheck
	}
}
