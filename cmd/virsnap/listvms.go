// Copyright (c) 2019 Jonas R. <joroec@gmx.net>
// Licensed under the MIT License. You have obtained a copy of the License at
// the "LICENSE" file in this repository.

// Package cmd implements the handlers for the different command line arguments.
package cmd

import (
  "fmt"
  
  "github.com/spf13/cobra"
  "github.com/libvirt/libvirt-go"
)

// listvmsCmd is a global variable defining the corresponding cobra command
var listvmsCmd = &cobra.Command{
  Use:   "listvms",
  Short: "List the virtual machines that can be detected via using libvirt.",
  Long:  "List the virtual machines that can be detected via using libvirt. "+
    "This is meant to be a simple method of testing your connection to the "+
    "libvirt daemon and should produce the same result as 'virsh list --all'.",
  Run: listvmsRun,
}

// init is a special golang function that is called exactly once regardless
// how often the package is imported.
func init() {
  // add command to root command so that cobra works as expected
  RootCmd.AddCommand(listvmsCmd)
}

// listvmsRun is the function called after the command line parser detected
// that we want to end up here. This functions connects to the libvirt daemon,
// retrieves the current list of virtual machines and prints it to standard
// output.
func listvmsRun(cmd *cobra.Command, args []string) {
  Logger.Trace("Start execution of listvmsRun function.")
  
  // Connecting to the QEMU socket
  Logger.Trace("Trying to connect to QEMU socket...")
  conn, err := libvirt.NewConnect("qemu:///system")
  if err != nil {
    Logger.Error("Could not connect to QEMU socket. Exiting")
    Logger.Fatal("Eror %s\n", err)
  }
  defer conn.Close()
  Logger.Trace("Successfully connected to QEMU socket.")
  
  // Retrieve all domains
  // The parameter for ListAllDomains is a bitmask that is used for filtering
  // the results. Since we do not want to restrict the usage to any strict type,
  // we use 0 which returns all of the found domains.
  Logger.Trace("Trying to retrieve all domains...")
  domains, err := conn.ListAllDomains(0)
  if err != nil {
    Logger.Error("Could not retrieve the QEMU domains. Exiting...")
    Logger.Fatal("Eror %s\n", err)
  }
  Logger.Trace("Successfully retrieved the QEMU domains.")
  
  // Loop over the domains and print out the name of the virtual machine.
  for _, domain := range domains {
    
    // This anonymous functions makes sure any single domain gets freed after
    // returning from the anonymous function as one does not want to use
    // 'defer' in a loop. To free a domain means to release the dynamic
    // memory allocated for the returned object. It does not mean to
    // undefine the VM or anything comparable.
    func(){
      defer domain.Free()
      
      Logger.Trace("Trying to retrieve a name for the domain...")
      name, err := domain.GetName()
      if err != nil {
        Logger.Error("Could not get the name of a domain. Skipping...")
        Logger.Error("Error: %s\n", err)
        return // no continue here, since we are in a anonymous function
      }
      Logger.Trace("Successfully retrieved the name of a domain.")
      
      fmt.Println(name)
    }()
  }
  
  Logger.Trace("Returning from listvmsRun function.")
}
