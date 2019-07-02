// Copyright (c) 2019 Jonas R. <joroec@gmx.net>
// Licensed under the MIT License. You have obtained a copy of the License at
// the "LICENSE" file in this repository.

// Package cmd implements the handlers for the different command line arguments.
package cmd

import (
  "fmt"

  "github.com/spf13/cobra"
)

// versionCmd is a global variable defining the corresponding cobra command
var versionCmd = &cobra.Command{
  Use:   "version",
  Short: "Prints the version of the software.",
  Long:  `Prints the version of the software.`,
  Run: versionRun,
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
  Logger.Trace("Start execution of versionRun function.")
  
  fmt.Println("virsnap, version 0.1.0")
  
  Logger.Trace("Returning from versionRun function.")
}
