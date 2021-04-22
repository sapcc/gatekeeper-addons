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
	"embed"
	"net/http"
	"os"
	"strconv"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/utils/openstack/clientconfig"
	"github.com/majewsky/schwift/gopherschwift"
	"github.com/sapcc/go-bits/httpee"
	"github.com/sapcc/go-bits/logg"
)

//go:embed static
var staticContent embed.FS

func main() {
	logg.ShowDebug, _ = strconv.ParseBool(os.Getenv("DOOP_CENTRAL_DEBUG"))
	if len(os.Args) != 2 {
		logg.Fatal("usage: %s <listen-address>", os.Args[0])
	}

	//initialize OpenStack/Swift client
	provider, err := clientconfig.AuthenticatedClient(nil)
	must("initialize OpenStack client", err)
	client, err := openstack.NewObjectStorageV1(provider, gophercloud.EndpointOpts{})
	must("initialize Swift client", err)
	account, err := gopherschwift.Wrap(client, &gopherschwift.Options{
		UserAgent: "doop-agent/rolling",
	})
	must("initialize Swift account", err)
	swiftContainer, err := account.Container(mustGetenv("REPORT_CONTAINER_NAME")).EnsureExists()
	must("initialize Swift container", err)

	//collect HTTP handlers
	mux := http.NewServeMux()
	mux.HandleFunc("/healthcheck", handleHealthcheck)
	mux.Handle("/static/", http.FileServer(http.FS(staticContent)))
	mux.HandleFunc("/", UI{NewDownloader(swiftContainer)}.RenderMainPage)

	//start HTTP server
	handler := logg.Middleware{}.Wrap(mux)
	logg.Info("listening on " + os.Args[1])
	ctx := httpee.ContextWithSIGINT(context.Background())
	err = httpee.ListenAndServeContext(ctx, os.Args[1], handler)
	if err != nil {
		logg.Fatal(err.Error())
	}
}

func must(task string, err error) {
	if err != nil {
		logg.Fatal("could not %s: %s", task, err.Error())
	}
}

func mustGetenv(key string) string {
	val := os.Getenv(key)
	if val == "" {
		logg.Fatal("missing required environment variable: " + key)
	}
	return val
}

func handleHealthcheck(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/healthcheck" && (r.Method == "GET" || r.Method == "HEAD") {
		http.Error(w, "OK", http.StatusOK)
	} else {
		http.Error(w, "not found", http.StatusNotFound)
	}
}
