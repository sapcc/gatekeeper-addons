/*******************************************************************************
*
* Copyright 2023 SAP SE
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
	"os"
	"time"

	"github.com/sapcc/go-bits/httpext"
	"github.com/sapcc/go-bits/logg"
	"github.com/sapcc/go-bits/must"
	"github.com/sapcc/go-bits/osext"
	"go.uber.org/automaxprocs/maxprocs"
)

func usage() {
	fmt.Fprintf(os.Stderr, "usage: %s [run|collect-once|analyze-once] <config-file>\n", os.Args[0])
	os.Exit(1)
}

func main() {
	logg.ShowDebug = osext.GetenvBool("DOOP_ANALYZER_DEBUG")
	undoMaxprocs, err := maxprocs.Set(maxprocs.Logger(logg.Debug))
	if err != nil {
		logg.Fatal("cannot setup GOMAXPROCS: " + err.Error())
	}
	defer undoMaxprocs()

	ctx := httpext.ContextWithSIGINT(context.Background(), 10*time.Second)
	if len(os.Args) != 3 {
		usage()
	}
	switch os.Args[1] {
	case "run":
		taskRun(ctx, os.Args[2])
	case "collect-once":
		taskCollectOnce(ctx, os.Args[2])
	case "analyze-once":
		taskAnalyzeOnce(ctx, os.Args[2])
	}
}

func taskRun(ctx context.Context, configPath string) {
	panic("TODO: unimplemented")
}

func taskCollectOnce(ctx context.Context, configPath string) {
	cfg := must.Return(ReadConfiguration(configPath))
	cs := must.Return(NewClientSet(cfg))
	report := must.Return(GatherReport(ctx, cfg, cs))
	buf := must.Return(json.MarshalIndent(report, "", "  "))
	_ = must.Return(os.Stdout.Write(buf))
}

func taskAnalyzeOnce(ctx context.Context, configPath string) {
	panic("TODO: unimplemented")
}
