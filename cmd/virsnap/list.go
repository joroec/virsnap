// Copyright (c) 2019 Jonas R. <joroec@gmx.net>
// Licensed under the MIT License. You have obtained a copy of the License at
// the "LICENSE" file in this repository.

// Package cmd implements the handlers for the different command line arguments.
package cmd

import (
  "fmt"
  "os"
  "time"
  "strconv"
  
  log "github.com/sirupsen/logrus"
  "github.com/spf13/cobra"
  "github.com/olekukonko/tablewriter"

  "github.com/joroec/virsnap/pkg/virt"
)


// listCmd is a global variable defining the corresponding cobra command
var listCmd = &cobra.Command{
  Use:   "list [<regex1>] [<regex2>] [<regex3>] ...",
  Short: "List snapshots of one or more virtual machines.",
  Long:  "List the virtual machine with the snapshots that can be detected "+
    "via using libvirt. This is meant to be a simple method of getting an "+
    "overview of the current virtual machines and the corresponding "+
    "snapshots. It is possible to specify a regular expression that filters "+
    "the shwon virtual machines by name. For example, 'virsnap list \".*\"' "+
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
  var err error
  var vms []virt.VM
  
  if len(args) > 0 {
    log.Debug("Using regular expression specified as command line argument.")
    vms, err = virt.ListMatchingVMs(args)
  } else {
    // listvms should display any virtual machine found. So, we need to specify
    // a search regex that matches any virtual machine name.
    log.Debug("Using default regular expression '.*', since no regular "+
      "expression was specified as command line argument.")
    regex := []string{".*"}
    vms, err = virt.ListMatchingVMs(regex)
  }
  
  if err != nil {
    err = fmt.Errorf("Could not retrieve the virtual machines from libvirt: %v",
      err)
    log.Fatal(err)
  }
  
  defer virt.FreeVMs(vms)

  
  for _, vm := range(vms) {
    snapshots, err := vm.ListMatchingSnapshots([]string{".*"})
    if err != nil {
      log.Errorf("Could not retrieve the snapshots for the domain %s. "+
        "Skipping the domain.", vm.Descriptor.Name)
      continue
    }
    
    defer virt.FreeSnapshots(snapshots)

    fmt.Println("VM:", vm.Descriptor.Name)

    table := tablewriter.NewWriter(os.Stdout)
    table.SetHeader([]string{"Snapshot", "Time", "State"})
    table.SetRowLine(false)

    for _, snapshot := range(snapshots) {
      
      time_int, err := strconv.ParseInt(snapshot.Descriptor.CreationTime, 10, 
        64)
      if err != nil {
        log.Errorf("Could not convert the snapshot creation time of VM %s. "+
          "Skipping VM: %v", vm.Descriptor.Name, err)
        continue
      }
      time := time.Unix(time_int, 0)
      
      table.Append([]string{snapshot.Descriptor.Name, 
        time.Format("Mon Jan 2 15:04:05 MST 2006"), snapshot.Descriptor.State})
    }
    
    table.Render()
    fmt.Println("")
  }
  
  
}
