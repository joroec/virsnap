// Copyright (c) 2019 Jonas R. <joroec@gmx.net>
// Licensed under the MIT License. You have obtained a copy of the License at
// the "LICENSE" file in this repository.

// Package cmd implements the handlers for the different command line arguments.
package cmd

import (
  "github.com/spf13/cobra"
  
  log "github.com/sirupsen/logrus"
  "github.com/joroec/virsnap/pkg/virt"
)

var keepVersions int

// cleanCmd is a global variable defining the corresponding cobra command
var cleanCmd = &cobra.Command{
  Use:   "clean -k <keep> <regex1> [<regex2>] [<regex3>] ...",
  Short: "Removes deprecated snapshots from the system.",
  Long:  `Removes deprecated snapshots from the system.`,
  Args: cobra.MinimumNArgs(1),
  Run: cleanRun,
}

// init is a special golang function that is called exactly once regardless
// how often the package is imported.
func init() {
  // initialize flags and arguments needed for this command
  cleanCmd.Flags().IntVarP(&keepVersions, "keep", "k", 10, "Number of "+
    "version to keep before begin cleaning. (required)")
  cleanCmd.MarkFlagRequired("keep")

  // add command to root command so that cobra works as expected
  RootCmd.AddCommand(cleanCmd)
}


func cleanRun(cmd *cobra.Command, args []string) {
  
  vms, err := virt.ListMatchingVMs(args)
  if err != nil {
    log.Fatal("Could not retrieve the virtual machines")
  }
  
  defer virt.FreeVMs(vms)
  
  for _, vm := range(vms) {
    
    // iterate over the domains and clean the snapshots for each of it
    snapshots, err := vm.ListMatchingSnapshots([]string{".*"})
    if err != nil {
      log.Error("Could not get the snapshot for VM %s. Skipping this VM: %v",
        vm.Descriptor.Name, err)
      continue
    }
    
    defer virt.FreeSnapshots(snapshots)
    
    // TODO: insert quick check whether there are enough snapshots
    
    
    // iterate over the snapshot exceeding the k snapshots that should
    // remain
    for i := 0; i < len(snapshots)-keepVersions; i++ {
      log.Info("Removing snapshot", snapshots[i].Descriptor.Name, "of VM", 
        vm.Descriptor.Name)
      err = snapshots[i].Instance.Delete(0)
      if err != nil {
        log.Error("Could not remove snapshot %s of VM %s: %v", 
          snapshots[i].Descriptor.Name, vm.Descriptor.Name, err)
      }
    }
    
  }
  
}
