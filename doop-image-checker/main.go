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
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"os"
	"time"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/gorilla/mux"
	"github.com/sapcc/go-bits/httpapi"
	"github.com/sapcc/go-bits/httpext"
	"github.com/sapcc/go-bits/logg"
	"github.com/sapcc/go-bits/osext"
	"gopkg.in/yaml.v2"
)

func main() {
	argCount := len(os.Args)
	if !(argCount == 2 || argCount == 3) {
		logg.Fatal("usage: %s <listen-address> [response-config.yaml]", os.Args[0])
	}

	var config map[string]map[string]string

	if len(os.Args) == 3 {
		byte, err := os.ReadFile(os.Args[2])
		if err != nil {
			logg.Fatal(err.Error())
		}

		config = make(map[string]map[string]string)
		err = yaml.Unmarshal(byte, config)
		if err != nil {
			logg.Fatal(err.Error())
		}
	}

	logAllRequests := osext.GetenvBool("LOG_ALL_REQUESTS")
	apis := []httpapi.API{
		api{config},
		httpapi.HealthCheckAPI{},
	}
	if !logAllRequests {
		apis = append(apis, httpapi.WithoutLogging())
	}
	handler := httpapi.Compose(apis...)

	logg.Info("listening on " + os.Args[1])
	ctx := httpext.ContextWithSIGINT(context.Background(), 10*time.Second)
	err := httpext.ListenAndServeContext(ctx, os.Args[1], handler)
	if err != nil {
		logg.Fatal(err.Error())
	}
}

type api struct {
	config map[string]map[string]string
}

// AddTo implements the httpapi.API interface.
func (a api) AddTo(r *mux.Router) {
	r.Methods("GET").Path("/v1/headers").HandlerFunc(a.handleHeaders)
}

func (a api) handleHeaders(w http.ResponseWriter, r *http.Request) {
	//validate request format
	imageRefStr := r.URL.Query().Get("image")
	if imageRefStr == "" {
		http.Error(w, `missing "image" query parameter`, http.StatusBadRequest)
		return
	}

	//use cache if possible
	hdr, ok := checkHeaderCache(imageRefStr)
	if ok {
		respondWithHeaderJSON(w, hdr)
		return
	}

	//parse image reference
	ref, err := name.ParseReference(imageRefStr, name.WithDefaultTag("latest"))
	if err != nil {
		msg := fmt.Sprintf("while parsing image reference %q: %s", html.EscapeString(imageRefStr), err.Error())
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	var respHeader http.Header
	// if a response config file is provided always use that
	if len(a.config) != 0 {
		respHeader = make(http.Header)
		for header, value := range a.config[imageRefStr] {
			if header == "X-Keppel-Max-Layer-Created-At" || header == "X-Keppel-Min-Layer-Created-At" {
				duration, err := time.ParseDuration(value)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				clock := time.Now().Add(duration)
				respHeader.Set(header, fmt.Sprintf("%d", clock.Unix()))
			} else {
				respHeader.Set(header, value)
			}
		}
	} else {
		var hc headerCapturer
		//make request to Keppel's Registry API (we deliberately make a GET here in
		//order to explicitly update the image's last_pulled_at timestamp);
		//unfortunately we need to engage in some trickery to extract the
		//X-Keppel-Vulnerability-Status header
		_, err = remote.Image(ref, remote.WithTransport(&hc))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		respHeader = hc.Headers
	}

	//fill cache and return result
	fillHeaderCache(imageRefStr, respHeader)
	respondWithHeaderJSON(w, respHeader)
}

type headerCapturer struct {
	Headers http.Header
}

// RoundTrip implements the http.RoundTripper interface.
func (hc *headerCapturer) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := http.DefaultTransport.RoundTrip(req)
	if err == nil && resp.Header.Get("Docker-Content-Digest") != "" {
		hc.Headers = resp.Header
	}
	return resp, err
}

func respondWithHeaderJSON(w http.ResponseWriter, hdr http.Header) {
	data := make(map[string]string)
	for k, v := range hdr {
		data[http.CanonicalHeaderKey(k)] = v[0]
	}
	buf, _ := json.Marshal(data) //nolint:errcheck
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(buf) //nolint:errcheck
}
