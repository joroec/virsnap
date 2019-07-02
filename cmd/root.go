/*
* Copyright (c) 2019 Jonas R. <joroec@gmx.net>
* Licensed under the MIT License. You have obtained a copy of the License at the
* "LICENSE" file in this repository.
*/
package cmd

import (
  "github.com/spf13/cobra"
)

var Verbose bool

var RootCmd = &cobra.Command{
  Use:   "kvmsnap",
  Short: "kvmsnap is a small tool that eases the creation of KVM snapshots "+
         "for backup purposes",
  Long:  "kvmsnap is a small tool that eases the creation of KVM snapshots "+
         "for backup purposes",
}

func init() {
  RootCmd.PersistentFlags().BoolVarP(&Verbose, "verbose", "v", false, "verbose output")
}
