// Copyright (c) 2019 The virnsnap authors. See file "AUTHORS".
// Licensed under the MIT License. You have obtained a copy of the License at
// the "LICENSE" file in this repository.

// Package main implements the handlers for the different command line arguments.
package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/bclicn/color"
	"github.com/joroec/virsnap/pkg/virt"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

// listCmd is a global variable defining the corresponding cobra command
var listCmd = &cobra.Command{
	Use:   "list [<regex1>] [<regex2>] [<regex3>] ...",
	Short: "List snapshots of one or more virtual machines",
	Long: "List the virtual machine with the snapshots that can be detected " +
		"via using libvirt. This is meant to be a simple method of getting an " +
		"overview of the current virtual machines and the corresponding " +
		"snapshots. It is possible to specify a regular expression that filters " +
		"the shwon virtual machines by name. For example, 'virsnap list \".*\"' " +
		"prints all accessible virtual machines with the corresponding snapshots " +
		", whereas 'virsnap list \"testing\"' prints only virtual machines with " +
		"the corresponding snapshots whose name includes \"testing\". If no " +
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
		logger.Debug("Using regular expression specified as command line argument: %#v", args)
		vms, err = virt.ListMatchingVMs(logger, args, socketURL)
	} else {
		// listvms should display any virtual machine found. So, we need to specify
		// a search regex that matches any virtual machine name.
		logger.Debug("Using default regular expression '.*', since no regular " +
			"expression was specified as command line argument")
		regex := []string{".*"}
		vms, err = virt.ListMatchingVMs(logger, regex, socketURL)
	}

	if err != nil {
		err = fmt.Errorf("unable to retrieve virtual machines from libvirt: %s",
			err,
		)
		logger.Fatalf("%s", err)
	}

	defer virt.FreeVMs(logger, vms)

	if len(vms) == 0 {
		logger.Fatal(errNoVMsMatchingRegex)
	}

	// iterate over the VMs and output the gathered information
	for index, vm := range vms {
		vmstate, err := vm.GetCurrentStateString()
		if err != nil {
			logger.Errorf("unable to retrieve current state of VM %s: %s",
				vm.Descriptor.Name,
				err,
			)
		}

		snapshots, err := vm.ListMatchingSnapshots([]string{".*"})
		if err != nil {
			logger.Errorf("skipping domain '%s': unable to retrieve snapshots for said domain: %s",
				vm.Descriptor.Name,
				err,
			)
			continue
		}

		// anonymous function for safely calling defer
		func() {
			defer virt.FreeSnapshots(logger, snapshots)

			// print the VM header to stdout
			fmt.Printf("%s (current state: %s, %d snapshots total)\n",
				color.BGreen(vm.Descriptor.Name), vmstate,
				len(snapshots))

			// print no snapshot table if there are no snapshots for this VM
			if len(snapshots) == 0 {
				return // anonymous function for safely calling defer
			}

			table := tablewriter.NewWriter(os.Stdout)
			table.SetHeader([]string{"Snapshot", "Time", "State"})
			table.SetRowLine(false)

			for _, snapshot := range snapshots {

				// convert timestamp to human-readable format
				timeInt, err := strconv.ParseInt(snapshot.Descriptor.CreationTime, 10, 64)
				if err != nil {
					logger.Errorf("skipping VM '%s': unable to convert snapshot creation time of VM: %s",
						vm.Descriptor.Name,
						err,
					)
					return // anonymous function for safely calling defer
				}
				time := time.Unix(timeInt, 0)

				// append the table row for this snapshot
				table.Append([]string{snapshot.Descriptor.Name,
					time.Format("Mon Jan 2 15:04:05 MST 2006"), snapshot.Descriptor.State})
			}

			table.Render()

			// do not print a new line if we are the last VM
			if index != len(vms)-1 {
				fmt.Println("")
			}

		}()

	}
}
