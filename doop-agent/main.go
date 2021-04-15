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
	"flag"
	"net/http"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/utils/openstack/clientconfig"
	"github.com/majewsky/schwift/gopherschwift"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sapcc/go-bits/httpee"
	"github.com/sapcc/go-bits/logg"
	wsk "github.com/wercker/stern/kubernetes"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func main() {
	var flagKubeconfig = flag.String("kubeconfig", "", "path to kubeconfig (required when not running in cluster)")
	var flagContext = flag.String("context", "", "override default k8s context (optional)")
	var flagListenAddress = flag.String("listen", ":8080", "listen address for Prometheus metrics endpoint")
	var flagContainer = flag.String("container", "", "name of Swift container in which to upload report")
	var flagObject = flag.String("object", "", "object name with which report will be uploaded in Swift")
	flag.Parse()

	if *flagContainer == "" {
		logg.Fatal("missing required option: -container")
	}
	if *flagObject == "" {
		logg.Fatal("missing required option: -object")
	}

	//initialize OpenStack/Swift client
	provider, err := clientconfig.AuthenticatedClient(nil)
	must("initialize OpenStack client", err)
	client, err := openstack.NewObjectStorageV1(provider, gophercloud.EndpointOpts{})
	must("initialize Swift client", err)
	account, err := gopherschwift.Wrap(client, &gopherschwift.Options{
		UserAgent: "doop-agent/rolling",
	})
	must("initialize Schwift account", err)
	swiftObj := account.Container(*flagContainer).Object(*flagObject)

	//initialize Kubernetes client
	var clientset *kubernetes.Clientset
	if *flagKubeconfig != "" {
		clientConfig := wsk.NewClientConfig(*flagKubeconfig, *flagContext)
		clientset, err = wsk.NewClientSet(clientConfig)
		must("initialize Kubernetes client", err)
	} else {
		config, err := rest.InClusterConfig()
		must("build Kubernetes config", err)
		clientset, err = kubernetes.NewForConfig(config)
		must("initialize Kubernetes client", err)
	}

	_ = clientset //TODO
	_ = swiftObj  //TODO

	//start HTTP server for Prometheus metrics
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	logg.Info("listening on " + *flagListenAddress)
	ctx := httpee.ContextWithSIGINT(context.Background())
	err = httpee.ListenAndServeContext(ctx, *flagListenAddress, mux)
	if err != nil {
		logg.Fatal(err.Error())
	}
}

func must(task string, err error) {
	if err != nil {
		logg.Fatal("could not %s: %s", task, err.Error())
	}
}
