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
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"net/http"
	"time"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/utils/openstack/clientconfig"
	"github.com/majewsky/schwift"
	"github.com/majewsky/schwift/gopherschwift"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sapcc/go-api-declarations/bininfo"
	"github.com/sapcc/go-bits/httpext"
	"github.com/sapcc/go-bits/logg"
	"k8s.io/client-go/rest"

	k8sinternal "github.com/sapcc/gatekeeper-addons/internal/kubernetes"
)

var (
	metricLastSuccessfulReport = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "doop_agent_last_successful_report",
		Help: "UNIX timestamp in seconds when last report was submitted.",
	})
	metricReportDurationSecs = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "doop_agent_report_duration_secs",
		Help: "How long it took to collect and submit the last report, in seconds.",
	})
)

type clusterIdentity struct {
	Layer string `json:"layer"`
	Type  string `json:"type"`
}

func main() {
	prometheus.MustRegister(metricLastSuccessfulReport)
	prometheus.MustRegister(metricReportDurationSecs)

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

	if flag.NArg() != 2 {
		logg.Fatal("need exactly two positional arguments")
	}
	identity := clusterIdentity{
		Layer: flag.Arg(0),
		Type:  flag.Arg(1),
	}

	wrap := httpext.WrapTransport(&http.DefaultTransport)
	wrap.SetOverrideUserAgent(bininfo.Component(), bininfo.VersionOr("rolling"))

	//initialize OpenStack/Swift client
	provider, err := clientconfig.AuthenticatedClient(nil)
	must("initialize OpenStack client", err)
	client, err := openstack.NewObjectStorageV1(provider, gophercloud.EndpointOpts{})
	must("initialize Swift client", err)
	account, err := gopherschwift.Wrap(client, nil)
	must("initialize Swift account", err)
	swiftContainer, err := account.Container(*flagContainer).EnsureExists()
	must("initialize Swift container", err)
	swiftObj := swiftContainer.Object(*flagObject)

	//initialize Kubernetes client
	var clientConfig *rest.Config
	if *flagKubeconfig != "" {
		clientConfig, err = k8sinternal.NewClientConfig(*flagKubeconfig, *flagContext).ClientConfig()
	} else {
		clientConfig, err = rest.InClusterConfig()
	}
	must("build Kubernetes config", err)
	clientset := NewClientSet(clientConfig) //NOTE: not kubernetes.NewForConfig() !

	//start HTTP server for Prometheus metrics
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	ctx := httpext.ContextWithSIGINT(context.Background(), 1*time.Second)
	go must.Succeed(httpext.ListenAndServeContext(ctx, *flagListenAddress, mux))

	//send a report immediately, then once a minute
	SendReport(ctx, clientset, swiftObj, identity)
	ticker := time.NewTicker(1 * time.Minute)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			SendReport(ctx, clientset, swiftObj, identity)
		}
	}
}

func must(task string, err error) {
	if err != nil {
		logg.Fatal("could not %s: %s", task, err.Error())
	}
}

// Report is the data structure that we write into our report file.
type Report struct {
	Identity  clusterIdentity     `json:"identity"`
	Templates []ReportForTemplate `json:"templates"`
}

// ReportForTemplate appears in type Report.
type ReportForTemplate struct {
	Kind        string            `json:"kind"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	Configs     []ReportForConfig `json:"configs"`
}

// ReportForConfig appears in type ReportForTemplate.
type ReportForConfig struct {
	Name           string                `json:"name"`
	Labels         map[string]string     `json:"labels,omitempty"`
	Annotations    map[string]string     `json:"annotations,omitempty"`
	AuditTimestamp string                `json:"auditTimestamp"`
	Violations     []ConstraintViolation `json:"violations"`
}

// SendReport queries the Kubernetes API to prepare a Report, and uploads the report to Swift.
func SendReport(ctx context.Context, cs ClientSet, swiftObj *schwift.Object, identity clusterIdentity) {
	start := time.Now()

	//build report
	logg.Info("building report")
	r := Report{Identity: identity}
	for _, t := range cs.ListConstraintTemplates(ctx) {
		rt := ReportForTemplate{
			Kind:        t.Spec.CRD.Spec.Names.Kind,
			Labels:      t.Metadata.Labels,
			Annotations: t.Metadata.Annotations,
		}
		for _, c := range cs.ListConstraintConfigs(ctx, t) {
			rc := ReportForConfig{
				Name:           c.Metadata.Name,
				Labels:         c.Metadata.Labels,
				Annotations:    c.Metadata.Annotations,
				AuditTimestamp: c.Status.AuditTimestamp,
				Violations:     c.Status.Violations,
			}
			rt.Configs = append(rt.Configs, rc)
		}
		r.Templates = append(r.Templates, rt)
	}

	//upload report
	logg.Info("uploading report")
	reportBytes, err := json.Marshal(r)
	must("encode report as JSON", err)
	err = swiftObj.Upload(bytes.NewReader(reportBytes), nil, nil)
	must("upload report to Swift", err)

	end := time.Now()
	duration := end.Sub(start)
	metricLastSuccessfulReport.Set(float64(end.Unix()))
	metricReportDurationSecs.Set(duration.Seconds())
	logg.Info("report submitted in %g seconds", duration.Seconds())
}
