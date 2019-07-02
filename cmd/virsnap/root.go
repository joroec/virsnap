// Copyright (c) 2019 Jonas R. <joroec@gmx.net>
// Licensed under the MIT License. You have obtained a copy of the License at
// the "LICENSE" file in this repository.

// Package cmd implements the handlers for the different command line arguments.
package cmd

import (
  "github.com/spf13/cobra"
  log "github.com/sirupsen/logrus"
)

// Verbose is a persistent flag that can be issued for any command issued over
// the command line.
var Verbose bool

// RootCmd is a global variable defining the corresponding cobra command.
// PersistentPreRun functions are inherited to child commands, so any command
// initializes the Logger accordingly if the verbose flag is set.
var RootCmd = &cobra.Command{
  Use:   "virsnap",
  Short: "virsnap is a small tool that eases the creation of KVM snapshots "+
         "for backup purposes",
  Long:  "virsnap is a small tool that eases the creation of KVM snapshots "+
         "for backup purposes",
  PersistentPreRun: initializeLogger,
}

// initializeLogger is a little helper function that enables tracing and
// debug outputs if the verbose flag is set.
func initializeLogger(cmd *cobra.Command, args []string) {
  if Verbose {
    log.SetLevel(log.TraceLevel)
  }
}

// init is a special golang function that is called exactly once regardless
// how often the package is imported.
func init() {
  RootCmd.PersistentFlags().BoolVarP(&Verbose, "verbose", "v", false, 
    "verbose output")
}
