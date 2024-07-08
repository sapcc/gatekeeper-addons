/*******************************************************************************
*
* Copyright 2022 SAP SE
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
	"io"
	"os"

	"github.com/sapcc/go-api-declarations/bininfo"
	"github.com/sapcc/go-bits/must"
	"gopkg.in/yaml.v3"

	"github.com/sapcc/gatekeeper-addons/internal/helmv3"
)

func main() {
	bininfo.HandleVersionArgument()

	buf := must.Return(io.ReadAll(os.Stdin))
	var contents helmv3.ReleaseContents
	must.Succeed(yaml.Unmarshal(buf, &contents))

	secretObj := must.Return(contents.GenerateMockRelease())
	buf = must.Return(yaml.Marshal(secretObj))
	_ = must.Return(os.Stdout.Write(buf))
}
