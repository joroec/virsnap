// Copyright (c) 2019 The virnsnap authors. See file "AUTHORS".
// Licensed under the MIT License. You have obtained a copy of the License at
// the "LICENSE" file in this repository.

// Package main implements the handlers for the different command line arguments.
package main

import (
	"fmt"
	"os"

	"github.com/joroec/virsnap/pkg/instrument/log"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var (
	// RootCmd is a global variable defining the corresponding cobra command.
	RootCmd = &cobra.Command{
		Use: "virsnap",
		Short: "virsnap is a small tool that eases the automated creation and " +
			"deletion of VM snapshots.",
		Long: "virsnap is a small tool that eases the automated creation and " +
			"deletion of VM snapshots.",
		PersistentPreRun: initLogger,
	}

	logger      *zap.SugaredLogger
	logLevel    = "info"
	logEncoding = "console"
	socketURL   = "qemu:///system"
)

// initLogger initializes a logger according to provided flags or their default
// values. This needs to be run as PersistenPreRun since those values
// need to be set when application is started, not when the package is imported
// (thus it can't be part of init()).
func initLogger(cmd *cobra.Command, args []string) {
	cfg := log.Configuration{
		Level:    logLevel,
		Encoding: logEncoding,
	}
	l, err := cfg.NewLogger()
	if err != nil {
		fmt.Printf("unable to initialize logger: %s\n", err)
		os.Exit(1)
	}

	logger = l.Sugar()
	logger.Debugf("Logger initialized")
}

// Execute runs the RootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// init is a special golang function that is called exactly once regardless
// how often the package is imported.
func init() {
	f := RootCmd.PersistentFlags()
	f.StringVarP(&logLevel, "log-level", "l", logLevel, "sets the log level (debug, info, warn, error)")
	f.StringVarP(&logEncoding, "log-encoding", "e", logEncoding, "sets the log encoding (console, json)")
	f.StringVarP(&socketURL, "socket-url", "u", socketURL, "sets the libvirt socket URL to connect to")
}
