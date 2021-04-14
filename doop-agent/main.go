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

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/utils/openstack/clientconfig"
	"github.com/majewsky/schwift/gopherschwift"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sapcc/go-bits/httpee"
	"github.com/sapcc/go-bits/logg"
)

func main() {
	if len(os.Args) != 2 {
		logg.Fatal("usage: %s <listen-address>", os.Args[0])
	}

	provider, err := clientconfig.AuthenticatedClient(nil)
	must("initialize OpenStack client", err)
	client, err := openstack.NewObjectStorageV1(provider, gophercloud.EndpointOpts{})
	must("initialize Swift client", err)
	account, err := gopherschwift.Wrap(client, nil)
	must("initialize Schwift account", err)
	swiftObj := account.Container(mustGetenv("REPORT_CONTAINER_NAME")).Object(mustGetenv("REPORT_OBJECT_NAME"))

	_ = swiftObj

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	logg.Info("listening on " + os.Args[1])
	ctx := httpee.ContextWithSIGINT(context.Background())
	err = httpee.ListenAndServeContext(ctx, os.Args[1], mux)
	if err != nil {
		logg.Fatal(err.Error())
	}
}

func mustGetenv(key string) string {
	val := os.Getenv(key)
	if val == "" {
		logg.Fatal("missing required environment variable: " + key)
	}
	return val
}

func must(task string, err error) {
	if err != nil {
		logg.Fatal("could not %s: %s", task, err.Error())
	}
}
