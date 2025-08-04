// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/sapcc/go-api-declarations/bininfo"
	"github.com/sapcc/go-bits/httpapi"
	"github.com/sapcc/go-bits/httpext"
	"github.com/sapcc/go-bits/logg"
	"github.com/sapcc/go-bits/must"
	"github.com/sapcc/go-bits/osext"
	"go.uber.org/automaxprocs/maxprocs"

	"github.com/sapcc/gatekeeper-addons/internal/helmv3"
)

func main() {
	bininfo.HandleVersionArgument()
	logg.ShowDebug = osext.GetenvBool("HELM_MANIFEST_PARSER_DEBUG")
	undoMaxprocs := must.Return(maxprocs.Set(maxprocs.Logger(logg.Debug)))
	defer undoMaxprocs()

	if len(os.Args) != 2 {
		logg.Fatal("usage: %s <listen-address>", os.Args[0])
	}

	logAllRequests := osext.GetenvBool("LOG_ALL_REQUESTS")
	apis := []httpapi.API{
		api{logAllRequests},
		httpapi.HealthCheckAPI{SkipRequestLog: true},
		// Even though the request handler limits the request payload size to 4 MiB,
		// we were seeing oomkills in prod on maxmemory = 128 MiB.
		// In all likelihood, this is because of too many requests in flight at once.
		httpapi.WithGlobalMiddleware(httpext.LimitConcurrentRequestsMiddleware(4)),
	}
	handler := httpapi.Compose(apis...)

	// during unit tests, we can set FAST_SHUTDOWN to avoid unnecessary waiting times
	shutdownDelay := 10 * time.Second
	if osext.GetenvBool("FAST_SHUTDOWN") {
		shutdownDelay = 100 * time.Millisecond
	}

	ctx := httpext.ContextWithSIGINT(context.Background(), shutdownDelay)
	must.Succeed(httpext.ListenAndServeContext(ctx, os.Args[1], handler))
}

type api struct {
	LogAllRequests bool
}

func (a api) AddTo(r *mux.Router) {
	r.Methods("POST").Path("/v3").HandlerFunc(a.handleAPI("/v3", helm3parse))
}

func (a api) handleAPI(path string, parser func([]byte) (string, error)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != path {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "only POST requests are allowed", http.StatusMethodNotAllowed)
			return
		}

		// never read more than 4 MiB to avoid DoS
		in, err := io.ReadAll(io.LimitReader(r.Body, 4<<20))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		out, err := parser(in)
		if err != nil {
			logg.Error("HTTP 400: " + err.Error())
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// HTTP 200 responses are usually silent to avoid useless log spam (but 4xx/5xx responses are always logged)
		if !a.LogAllRequests {
			httpapi.SkipRequestLog(r)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(out)) //nolint:errcheck
	}
}

func helm3parse(in []byte) (string, error) {
	result, err := helmv3.ParseRelease(in)
	if err != nil {
		return "", err
	}
	out, err := json.Marshal(result)
	return string(out), err
}
