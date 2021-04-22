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
	"encoding/json"
	"net/http"
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

	//TODO build actual report page
	result := make(map[string]int)
	for name, report := range reports {
		result[name] = len(report)
	}
	resultBytes, _ := json.Marshal(result)
	http.Error(w, string(resultBytes), http.StatusOK)
}
