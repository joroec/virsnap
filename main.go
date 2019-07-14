// Copyright (c) 2019 The virnsnap authors. See file "AUTHORS".
// Licensed under the MIT License. You have obtained a copy of the License at
// the "LICENSE" file in this repository.

// Package main is the top-level package of the application.
package main

import (
	"github.com/joroec/virsnap/cmd/virsnap"
)

// main is the entry point of the application. Just pass execution to command
// line parser.
func main() {
	cmd.RootCmd.Execute()
}
