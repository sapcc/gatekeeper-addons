// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sapcc/go-api-declarations/bininfo"
	"github.com/sapcc/go-bits/httpext"
	"github.com/sapcc/go-bits/logg"
	"github.com/sapcc/go-bits/must"
	"github.com/sapcc/go-bits/osext"
	"go.uber.org/automaxprocs/maxprocs"

	"github.com/sapcc/gatekeeper-addons/internal/doop"
)

func usage() {
	fmt.Fprintf(os.Stderr, "usage: %s [run|collect-once|process-once] <config-file>\n", os.Args[0])
	os.Exit(1)
}

func main() {
	bininfo.HandleVersionArgument()
	logg.ShowDebug = osext.GetenvBool("DOOP_ANALYZER_DEBUG")
	undoMaxprocs := must.Return(maxprocs.Set(maxprocs.Logger(logg.Debug)))
	defer undoMaxprocs()

	wrap := httpext.WrapTransport(&http.DefaultTransport)
	wrap.SetOverrideUserAgent(bininfo.Component(), bininfo.VersionOr("rolling"))

	ctx := httpext.ContextWithSIGINT(context.Background(), 1*time.Second)
	if len(os.Args) != 3 {
		usage()
	}
	switch os.Args[1] {
	case "run":
		taskRun(ctx, os.Args[2])
	case "collect-once":
		taskCollectOnce(ctx, os.Args[2])
	case "process-once":
		taskProcessOnce(ctx, os.Args[2])
	default:
		usage()
	}
}

var (
	metricLastSuccessfulReport = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "doop_analyzer_last_successful_report",
		Help: "UNIX timestamp in seconds when last report was submitted.",
	})
	metricReportDurationSecs = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "doop_analyzer_report_duration_secs",
		Help: "How long it took to collect and submit the last report, in seconds.",
	})
)

func taskRun(ctx context.Context, configPath string) {
	prometheus.MustRegister(metricLastSuccessfulReport)
	prometheus.MustRegister(metricReportDurationSecs)

	cfg := must.Return(ReadConfiguration(configPath))
	cs := must.Return(NewClientSet(cfg))
	must.Succeed(cfg.Swift.Connect(ctx))

	// start HTTP server for Prometheus metrics
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	go func() {
		must.Succeed(httpext.ListenAndServeContext(ctx, cfg.Metrics.ListenAddress, mux))
	}()

	// send a report immediately, then once a minute
	sendReport(ctx, cfg, cs)
	ticker := time.NewTicker(1 * time.Minute)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sendReport(ctx, cfg, cs)
		}
	}
}

func sendReport(ctx context.Context, cfg Configuration, cs ClientSetInterface) {
	start := time.Now()

	report := must.Return(GatherReport(ctx, cfg, cs))
	ProcessReport(&report, cfg)
	must.Succeed(cfg.Swift.SendReport(ctx, report))

	end := time.Now()
	duration := end.Sub(start)
	metricLastSuccessfulReport.Set(float64(end.Unix()))
	metricReportDurationSecs.Set(duration.Seconds())
	logg.Info("report uploaded in %g seconds", duration.Seconds())
}

func taskCollectOnce(ctx context.Context, configPath string) {
	cfg := must.Return(ReadConfiguration(configPath))
	cs := must.Return(NewClientSet(cfg))
	report := must.Return(GatherReport(ctx, cfg, cs))
	printJSON(report)
}

func taskProcessOnce(_ context.Context, configPath string) {
	cfg := must.Return(ReadConfiguration(configPath))
	cfg.ValidateRules().LogFatalIfError()
	var report doop.Report
	must.Succeed(json.NewDecoder(os.Stdin).Decode(&report))
	ProcessReport(&report, cfg)
	printJSON(report)
}

func printJSON(data any) {
	writer := bufio.NewWriter(os.Stdout)
	enc := json.NewEncoder(writer)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	must.Succeed(enc.Encode(data))
	must.Succeed(writer.Flush())
}
