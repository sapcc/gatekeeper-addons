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
	_ "embed"
	"encoding/json"
	"net/http"
	"text/template"

	"github.com/sapcc/go-bits/logg"
)

//go:embed index.html.tpl
var pageTemplateStr string

var (
	pageTemplate = template.Must(template.New("index.html").Funcs(funcMap).Parse(pageTemplateStr))
	funcMap      = template.FuncMap{
		"jsonIndent": jsonIndent,
	}
)

//UI provides the business logic for rendering the web dashboard.
type UI struct {
	d *Downloader
}

//RenderMainPage is a http.HandleFunc for `GET /`.
func (ui UI) RenderMainPage(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" || (r.Method != "GET" && r.Method != "HEAD") {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	reports, err := ui.d.GetReports()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	//TODO preprocess `reports` before passing it into the template
	w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'self' 'unsafe-inline'")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	err = pageTemplate.Execute(w, reports)
	if err != nil {
		logg.Error("while rendering index.html: %s", err.Error())
	}
}

func jsonIndent(in []byte) string {
	var buf bytes.Buffer
	json.Indent(&buf, in, "", "  ")
	return buf.String()
}
