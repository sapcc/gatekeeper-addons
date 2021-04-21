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
	"net/http"
	"os"

	"github.com/sapcc/go-bits/httpee"
	"github.com/sapcc/go-bits/logg"
)

func main() {
	if len(os.Args) != 2 {
		logg.Fatal("usage: %s <listen-address>", os.Args[0])
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthcheck", handleHealthcheck)

	handler := logg.Middleware{}.Wrap(mux)

	logg.Info("listening on " + os.Args[1])
	ctx := httpee.ContextWithSIGINT(context.Background())
	err := httpee.ListenAndServeContext(ctx, os.Args[1], handler)
	if err != nil {
		logg.Fatal(err.Error())
	}
}

func handleHealthcheck(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/healthcheck" && (r.Method == "GET" || r.Method == "HEAD") {
		http.Error(w, "OK", http.StatusOK)
	} else {
		http.Error(w, "not found", http.StatusNotFound)
	}
}
