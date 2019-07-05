// Copyright (c) 2019 Jonas R. <joroec@gmx.net>
// Licensed under the MIT License. You have obtained a copy of the License at
// the "LICENSE" file in this repository.

// Package cmd implements the handlers for the different command line arguments.
package cmd

import (
  "fmt"
  
  log "github.com/sirupsen/logrus"
  "github.com/spf13/cobra"

  "github.com/joroec/virsnap/pkg/vm"
)


// listCmd is a global variable defining the corresponding cobra command
var listCmd = &cobra.Command{
  Use:   "list [<regex1>] [<regex2>] [<regex3>] ...",
  Short: "List snapshots of one or more virtual machines.",
  Long:  "List the virtual machine with the snapshots that can be detected "+
    "via using libvirt. This is meant to be a simple method of getting an "+
    "overview of the current virtual machines and the corresponding "+
    "snapshots. It is possible to specify a regular expression that filters "+
    "the shwon virtual names by name. For example, 'virsnap list \".*\"' "+
    "prints all accessible virtual machines with the corresponding snapshots "+
    ", whereas 'virsnap list \"testing\"' prints only virtual machines with "+
    "the corresponding snapshots whose name includes \"testing\". If no "+
    "regex is given, any acccessible virtual machine is printed.",
  Run: listRun,
}

// init is a special golang function that is called exactly once regardless
// how often the package is imported.
func init() {
  // add command to root command so that cobra works as expected
  RootCmd.AddCommand(listCmd)
}

// listRun is the function called after the command line parser detected
// that we want to end up here.
func listRun(cmd *cobra.Command, args []string) {
  log.Trace("Start execution of listRun function.")
  
  var err error
  var vms []vm.NamedVM
  if len(args) > 0 {
    // a regex has been specified, so we take it to filter the virtual machines
    vms, err = vm.GetMatchingVMs(args)
  } else {
    // listvms should display any virtual machine found. So, we need to specify
    // a search regex that matches any virtual machine name.
    regex := []string{".*"}
    vms, err = vm.GetMatchingVMs(regex)
  }
  
  if err != nil {
    log.Error("Could not retrieve the virtual machines")
    return
  }
  
  defer vm.FreeVMs(vms)
  
  for _, vm := range(vms) {
    fmt.Println(vm.Name)
    
    // since we do not want to filter the results, we pass a 0 as argument
    snapshots, err := vm.Domain.ListAllSnapshots(0)
    if err != nil {
      log.Error("Could not retrieve the snapshots for the domain:", 
        vm.Name, ". Skipping the domain.")
      continue
    }
    
    // iterate over the snapshots, retrieve the name and print it
    for _, snapshot := range(snapshots) {
    
      func(){
        defer snapshot.Free()
        
        name, err := snapshot.GetName()
        if err != nil {
          log.Error("Could not get the name of a snapshot. Skipping...")
          log.Error("Error: %s\n", err)
          return // no continue here, since we are in an anonymous function
        }
        
        fmt.Println("    "+name)
      }()
    }
  }
}
