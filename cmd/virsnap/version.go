// Copyright (c) 2019 The virnsnap authors. See file "AUTHORS".
// Licensed under the MIT License. You have obtained a copy of the License at
// the "LICENSE" file in this repository.

package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

const (
	version = "0.1.0"
)

// versionCmd is a global variable defining the corresponding cobra command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version of the software",
	Long:  `Print the version of the software.`,
	Run:   versionRun,
}

// init is a special golang function that is called exactly once regardless
// how often the package is imported.
func init() {
	RootCmd.AddCommand(versionCmd)
}

// versionRun is the function called after the command line parser detected
// that we want to end up here. This functions does noting except printing
// the current version number.
func versionRun(cmd *cobra.Command, args []string) {
	fmt.Printf("virsnap, version %s\n", version)
}
