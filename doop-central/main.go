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
	"github.com/sapcc/go-api-declarations/bininfo"
	"github.com/sapcc/go-bits/httpapi"
	"github.com/sapcc/go-bits/httpext"
	"github.com/sapcc/go-bits/logg"
	"github.com/sapcc/go-bits/must"
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

	wrap := httpext.WrapTransport(&http.DefaultTransport)
	wrap.SetOverrideUserAgent(bininfo.Component(), bininfo.VersionOr("rolling"))

	//parse docs.yaml
	docstringsBytes := must.Return(os.ReadFile(os.Args[2]))
	docstrings := make(map[string]template.HTML)
	must.Succeed(yaml.Unmarshal(docstringsBytes, &docstrings))

	//initialize OpenStack/Swift client
	ao := must.Return(clientconfig.AuthOptions(nil))
	ao.AllowReauth = true
	provider := must.Return(openstack.NewClient(ao.IdentityEndpoint))
	must.Succeed(openstack.Authenticate(provider, *ao))
	client := must.Return(openstack.NewObjectStorageV1(provider, gophercloud.EndpointOpts{}))
	account := must.Return(gopherschwift.Wrap(client, nil))
	swiftContainer := must.Return(account.Container(osext.MustGetenv("REPORT_CONTAINER_NAME")).EnsureExists())

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
	ctx := httpext.ContextWithSIGINT(context.Background(), 10*time.Second)
	must.Succeed(httpext.ListenAndServeContext(ctx, os.Args[1], nil))
}
