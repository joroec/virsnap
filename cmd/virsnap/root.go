// Copyright (c) 2019 Jonas R. <joroec@gmx.net>
// Licensed under the MIT License. You have obtained a copy of the License at
// the "LICENSE" file in this repository.

// Package cmd implements the handlers for the different command line arguments.
package cmd

import (
  "os"

  "github.com/Mandala/go-log"
  "github.com/spf13/cobra"
)

// Verbose is a persistent flag that can be issued for any command issued over
// the command line.
var Verbose bool

// Logger is the global logging object that can be used for console output.
var Logger *log.Logger

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
    Logger.WithDebug()
  }
}

// init is a special golang function that is called exactly once regardless
// how often the package is imported.
func init() {
  Logger = log.New(os.Stdout)

  RootCmd.PersistentFlags().BoolVarP(&Verbose, "verbose", "v", false, 
    "verbose output")
}
