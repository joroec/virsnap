package cmd
/*
* Copyright (c) 2019 Jonas R. <joroec@gmx.net>
* Licensed under the MIT License. You have obtained a copy of the License at the
* "LICENSE" file in this repository.
*/

import (
  "fmt"

  "github.com/spf13/cobra"
)

func init() {
  RootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
  Use:   "version",
  Short: "Prints the version of the software.",
  Long:  `Prints the version of the software.`,
  Run: versionRun,
}

func versionRun(cmd *cobra.Command, args []string) {
    fmt.Println("kvmsnap, version 0.1.0")
}
