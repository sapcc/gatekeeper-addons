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
	"html/template"
	"net/http"
	"os"
	"time"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/utils/openstack/clientconfig"
	"github.com/majewsky/schwift/gopherschwift"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sapcc/go-bits/httpapi"
	"github.com/sapcc/go-bits/httpext"
	"github.com/sapcc/go-bits/logg"
	"github.com/sapcc/go-bits/osext"
	"gopkg.in/yaml.v2"
)

//go:embed static
var staticContent embed.FS

func main() {
	logg.ShowDebug = osext.GetenvBool("DOOP_CENTRAL_DEBUG")
	if len(os.Args) != 3 {
		logg.Fatal("usage: %s <listen-address> <docs.yaml>", os.Args[0])
	}

	//parse docs.yaml
	docstringsBytes, err := os.ReadFile(os.Args[2])
	must("read docstring file", err)
	docstrings := make(map[string]template.HTML)
	err = yaml.Unmarshal(docstringsBytes, &docstrings)
	must("parse docstring file", err)

	//initialize OpenStack/Swift client
	ao, err := clientconfig.AuthOptions(nil)
	must("find OpenStack credentials", err)
	ao.AllowReauth = true
	provider, err := openstack.NewClient(ao.IdentityEndpoint)
	must("initialize OpenStack client", err)
	err = openstack.Authenticate(provider, *ao)
	must("initialize OpenStack authentication", err)
	client, err := openstack.NewObjectStorageV1(provider, gophercloud.EndpointOpts{})
	must("initialize Swift client", err)
	account, err := gopherschwift.Wrap(client, &gopherschwift.Options{
		UserAgent: "doop-central/rolling",
	})
	must("initialize Swift account", err)
	swiftContainer, err := account.Container(osext.MustGetenv("REPORT_CONTAINER_NAME")).EnsureExists()
	must("initialize Swift container", err)

	//collect HTTP handlers
	ui := UI{NewDownloader(swiftContainer), docstrings}
	prometheus.MustRegister(ui)
	handler := httpapi.Compose(
		ui,
		httpapi.HealthCheckAPI{SkipRequestLog: true},
	)
	http.Handle("/", handler)
	http.Handle("/metrics", promhttp.Handler())
	http.Handle("/static/", http.FileServer(http.FS(staticContent)))

	//start HTTP server
	logg.Info("listening on " + os.Args[1])
	ctx := httpext.ContextWithSIGINT(context.Background(), 10*time.Second)
	err = httpext.ListenAndServeContext(ctx, os.Args[1], nil)
	if err != nil {
		logg.Fatal(err.Error())
	}
}

func must(task string, err error) {
	if err != nil {
		logg.Fatal("could not %s: %s", task, err.Error())
	}
}
