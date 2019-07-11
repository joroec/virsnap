// Copyright (c) 2019 The virnsnap authors. See file "AUTHORS".
// Licensed under the MIT License. You have obtained a copy of the License at
// the "LICENSE" file in this repository.

// Package cmd implements the handlers for the different command line arguments.
package cmd

import (
  "bufio"
  "fmt"
  "os"
  "strings"
  
  "github.com/spf13/cobra"
  
  "github.com/joroec/virsnap/pkg/virt"
  log "github.com/sirupsen/logrus"
)

// keepVersions is a global variable determing how many successive snapshot
// of the VM should be kept before beginning to remove the old ones.
var keepVersions int

// assumeYes is a global variable determing whether snapshots should be deleted
// without additional confirmation.
var assumeYes bool

// cleanCmd is a global variable defining the corresponding cobra command
var cleanCmd = &cobra.Command{
  Use:   "clean [-y] -k <keep> <regex1> [<regex2>] [<regex3>] ...",
  Short: "Remove expired snapshots from the system",
  Long:  "Remove expired snapshots from the system. The parameter k "+
    "specifies how many successive snapshots of a VM should be kept before "+
    "beginning to remove the old ones. For example, if you take a snapshot "+
    "every day over a time of 30 days and then call clean with an k of 15, "+
    "the oldest 15 snapshots that were created by libvirt get removed. The "+
    "regular expression(s) determine the VM names of those VM whose "+
    "snapshots should get cleaned. For example, 'virsnap clean -k 10 \".*\"' "+
    "cleans the snapshots of all found virtual machines, whereas "+
    "'virsnap clean -k 10 \"testing\"' cleans the snapshots only for those "+
    "virtial machines whose name includes \"testing\". ",
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

  cleanCmd.Flags().BoolVarP(&assumeYes, "assume-yes", "y", false, "Do not ask "+
    "for additional confirmation when about to remove a snapshot. Useful for "+
    "automated execution.")

  // add command to root command so that cobra works as expected
  RootCmd.AddCommand(cleanCmd)
}

// cleanRun takes as parameter the name of the VMs to clean
func cleanRun(cmd *cobra.Command, args []string) {
  // check the validity of the console line parameters
  if keepVersions < 0 {
    log.Fatal("The parameter k must not be negative!")
  }
  
  vms, err := virt.ListMatchingVMs(args)
  if err != nil {
    log.Fatal("Could not retrieve the virtual machines.")
  }
  
  defer virt.FreeVMs(vms)
  
  if len(vms) == 0 {
    log.Info("There were no virtual machines matchig the given regular "+
      "expression(s).")
    return
  }
  log.Debugf("Found %d matching VMs.", len(vms))
  
  if assumeYes {
    log.Debugf("Remove snapshots without any further confirmation.")
  }
  
  // a boolean indicating whether at least one error occured. Useful for
  // the exit code of the program after iterating over the virtual machines.
  failed := false
  
  for _, vm := range(vms) {
    
    // iterate over the domains and clean the snapshots for each of it
    snapshots, err := vm.ListMatchingSnapshots([]string{".*"})
    if err != nil {
      log.Errorf("Could not get the snapshot for VM \"%s\". Skipping this "+
        "VM: %v", vm.Descriptor.Name, err)
      failed = true
      continue
    }
    
    func(){ // anonymous function for not calling snapshot.Free in a loop
        defer virt.FreeSnapshots(snapshots)
        
        if len(snapshots) <= keepVersions {
          return
        }
        
        // iterate over the snapshot exceeding the k snapshots that should
        // remain
        for i := 0; i < len(snapshots)-keepVersions; i++ {
          log.Infof("About to remove snapshot \"%s\" of VM \"%s\".",
            snapshots[i].Descriptor.Name, vm.Descriptor.Name)
          
          accepted := false
          if assumeYes {
            accepted = true
          } else {
            accepted = confirm("Remove snapshot?", 10)
          }
          
          if accepted {
            log.Infof("Removing snapshot \"%s\" of VM \"%s\".",
              snapshots[i].Descriptor.Name, vm.Descriptor.Name)
            
            err = snapshots[i].Instance.Delete(0)
            if err != nil {
              log.Errorf("Could not remove snapshot \"%s\" of VM \"%s\". "+
                "Skipping this VM: %v", snapshots[i].Descriptor.Name,
                vm.Descriptor.Name, err)
              failed = true
              return // we are in an anonymous function
            }
          } else {
            log.Infof("Skipping removal of snapshot \"%s\" of VM \"%s\".",
              snapshots[i].Descriptor.Name, vm.Descriptor.Name)
          }
        }
    }()
  }
  
  if failed {
    log.Fatal("There were some errors during the clean process.")
  }
}

// confirm displays a prompt `s` to the user and returns a bool indicating
// yes / no. If the lowercased, trimmed input begins with anything other than
// 'y', it returns false. It accepts an int `tries` representing the number of
// attempts before returning false
func confirm(s string, tries int) bool {
	r := bufio.NewReader(os.Stdin)

	for ; tries > 0; tries-- {
		fmt.Printf("%s [y/n]: ", s)

		res, err := r.ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}

		// Empty input (i.e. "\n")
		if len(res) < 2 {
			continue
		}

		return strings.ToLower(strings.TrimSpace(res))[0] == 'y'
	}

	return false
}
