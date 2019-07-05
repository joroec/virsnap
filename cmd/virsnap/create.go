// Copyright (c) 2019 Jonas R. <joroec@gmx.net>
// Licensed under the MIT License. You have obtained a copy of the License at
// the "LICENSE" file in this repository.

// Package cmd implements the handlers for the different command line arguments.
package cmd

import (
  "github.com/spf13/cobra"
  log "github.com/sirupsen/logrus"
  "github.com/docker/docker/pkg/namesgenerator"

  "github.com/libvirt/libvirt-go"
  "github.com/libvirt/libvirt-go-xml"
  
  "github.com/joroec/virsnap/pkg/virt"
)

// a global variable determing whether virsnap should try to shutdown the
// virtual machine before taking the snapshot
var shutdown bool

// createCmd is a global variable defining the corresponding cobra command
// TODO: add some more command line usage documentation
// We just expect that the snapshot behavior for the disks is specified
// correctly.
var createCmd = &cobra.Command{
  Use:   "create <regex1> [<regex2>] [<regex3>] ...",
  Short: "Creates a snapshot of one or more KVM virtual machines",
  Long:  `Creates a snapshot of one or more KVM virtual machines`,
  Args: cobra.MinimumNArgs(1),
  Run: createRun,
}

// init is a special golang function that is called exactly once regardless
// how often the package is imported.
func init() {
  // initialize flags and arguments needed for this command
  createCmd.Flags().BoolVarP(&shutdown, "shutdown", "s", false, "Try to "+
    "shutdown the VM before making the snapshot. Restores state afterwards.")

  // add command to root command so that cobra works as expected
  RootCmd.AddCommand(createCmd)
}

// args are the name of the VMs to backup
func createRun(cmd *cobra.Command, args []string) {
  
  vms, err := virt.ListMatchingVMs(args)
  if err != nil {
    log.Fatal("Could not retrieve the virtual machines")
  }
  
  // TODO: remove anonymous function, since we have this function? Implement
  // the same for snapshots?
  defer virt.FreeVMs(vms)
  
  for _, vm := range(vms) {
    // iterate over the domains and crete a new snapshot for each of it
    
    former_state := libvirt.DOMAIN_NOSTATE
    if(shutdown) {
      former_state, err = vm.Shutdown()
      if err != nil {
        log.Error(err)
        continue
      }
    }
    
    log.Trace("Beginning creation of snapshot for VM:", vm.Descriptor.Name)
    
    descriptor := &libvirtxml.DomainSnapshot{
      Name: "virsnap_"+ namesgenerator.GetRandomName(0),
      Description: "snapshot created by virsnap.",
    }
    
    xml, err := descriptor.Marshal()
    if err != nil {
      log.Error("Could not marshal the snapshot xml for VM:",
        vm.Descriptor.Name, ". Skipping the VM.")
      return // we are in an anonymous function
    }
    
    snapshot, err := vm.Instance.CreateSnapshotXML(xml, 0)
    if err != nil {
      log.Error("Could not create the snapshot for the VM:", vm.Descriptor.Name,
        ". Skipping the VM.")
      return // we are in an anonymous function
    }
    defer snapshot.Free()
    
    if(former_state == libvirt.DOMAIN_RUNNING) {
      err = vm.Start()
      if err != nil {
        log.Error(err)
        return // we are in an anonymous function
      }
    }
    
    log.Trace("Leaving creation of snapshot for VM:", vm.Descriptor.Name)
  }
  
  log.Trace("Start execution of createRun function.")
}
