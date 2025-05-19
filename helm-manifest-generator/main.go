// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"io"
	"os"

	"github.com/sapcc/go-api-declarations/bininfo"
	"github.com/sapcc/go-bits/must"
	"gopkg.in/yaml.v2"

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
