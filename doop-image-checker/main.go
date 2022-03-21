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
	"strconv"
	"time"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/sapcc/go-bits/httpee"
	"github.com/sapcc/go-bits/logg"
)

func main() {
	if len(os.Args) != 2 {
		logg.Fatal("usage: %s <listen-address>", os.Args[0])
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthcheck", handleHealthcheck)
	mux.HandleFunc("/v1/headers", handleHeaders)

	handler := getLogMiddleware().Wrap(mux)

	logg.Info("listening on " + os.Args[1])
	ctx := httpee.ContextWithSIGINT(context.Background(), 10*time.Second)
	err := httpee.ListenAndServeContext(ctx, os.Args[1], handler)
	if err != nil {
		logg.Fatal(err.Error())
	}
}

func getLogMiddleware() logg.Middleware {
	logAllRequests, _ := strconv.ParseBool(os.Getenv("LOG_ALL_REQUESTS")) //nolint:errcheck
	if logAllRequests {
		return logg.Middleware{}
	}
	return logg.Middleware{ExceptStatusCodes: []int{200}}
}

func handleHealthcheck(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/healthcheck" && (r.Method == "GET" || r.Method == "HEAD") {
		http.Error(w, "OK", http.StatusOK)
	} else {
		http.Error(w, "not found", http.StatusNotFound)
	}
}

func handleHeaders(w http.ResponseWriter, r *http.Request) {
	//validate request format
	if r.URL.Path != "/v1/headers" {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if r.Method != "GET" {
		http.Error(w, "only GET requests are allowed", http.StatusMethodNotAllowed)
		return
	}
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

	//make request to Keppel's Registry API (we deliberately make a GET here in
	//order to explicitly update the image's last_pulled_at timestamp);
	//unfortunately we need to engage in some trickery to extract the
	//X-Keppel-Vulnerability-Status header
	var hc headerCapturer
	_, err = remote.Image(ref, remote.WithTransport(&hc))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	//fill cache and return result
	fillHeaderCache(imageRefStr, hc.Headers)
	respondWithHeaderJSON(w, hc.Headers)
}

type headerCapturer struct {
	Headers http.Header
}

//RoundTrip implements the http.RoundTripper interface.
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
