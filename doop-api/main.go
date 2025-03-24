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
	"context"
	"net/http"
	"time"

	"github.com/gophercloud/gophercloud/v2/openstack"
	"github.com/majewsky/schwift/v2/gopherschwift"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sapcc/go-api-declarations/bininfo"
	"github.com/sapcc/go-bits/gophercloudext"
	"github.com/sapcc/go-bits/httpapi"
	"github.com/sapcc/go-bits/httpapi/pprofapi"
	"github.com/sapcc/go-bits/httpext"
	"github.com/sapcc/go-bits/logg"
	"github.com/sapcc/go-bits/must"
	"github.com/sapcc/go-bits/osext"
	"go.uber.org/automaxprocs/maxprocs"
)

func main() {
	bininfo.HandleVersionArgument()
	logg.ShowDebug = osext.GetenvBool("DOOP_API_DEBUG")
	undoMaxprocs := must.Return(maxprocs.Set(maxprocs.Logger(logg.Debug)))
	defer undoMaxprocs()

	wrap := httpext.WrapTransport(&http.DefaultTransport)
	wrap.SetOverrideUserAgent(bininfo.Component(), bininfo.VersionOr("rolling"))

	ctx := httpext.ContextWithSIGINT(context.Background(), 10*time.Second)

	// initialize OpenStack/Swift client
	provider, eo, err := gophercloudext.NewProviderClient(ctx, nil)
	must.Succeed(err)
	provider.UserAgent.Prepend("doop-api")
	client := must.Return(openstack.NewObjectStorageV1(provider, eo))
	account := must.Return(gopherschwift.Wrap(client, nil))
	containerName := osext.MustGetenv("DOOP_API_SWIFT_CONTAINER")
	container := must.Return(account.Container(containerName).EnsureExists(ctx))
	downloader := NewDownloader(container)

	// collect HTTP handlers
	prometheus.MustRegister(NewMetricCollector(downloader))
	handler := httpapi.Compose(
		API{downloader},
		httpapi.HealthCheckAPI{SkipRequestLog: true},
		pprofapi.API{IsAuthorized: pprofapi.IsRequestFromLocalhost},
	)
	mux := http.NewServeMux()
	mux.Handle("/", handler)
	mux.Handle("/metrics", promhttp.Handler())

	// start HTTP server
	listenAddress := osext.GetenvOrDefault("DOOP_API_LISTEN_ADDRESS", ":8080")
	must.Succeed(httpext.ListenAndServeContext(ctx, listenAddress, mux))
}
