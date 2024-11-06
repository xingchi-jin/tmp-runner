// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the PolyForm Shield 1.0.0 license
// that can be found in the licenses directory at the root of this repository, also available at
// https://polyformproject.org/wp-content/uploads/2020/06/PolyForm-Shield-1.0.0.txt.

package cli

import (
	"os"

	"github.com/harness/runner/cli/server"
	"github.com/harness/runner/version"

	"gopkg.in/alecthomas/kingpin.v2"
)

// Command parses the command line arguments and then executes a
// subcommand program.
func Command() {
	app := kingpin.New("runner", "Harness Runner to execute tasks")
	app.HelpFlag.Short('h')
	app.Version(version.Version)
	app.VersionFlag.Short('v')
	server.Register(app)

	kingpin.MustParse(app.Parse(os.Args[1:]))
}
