// Copyright (c) 2019 The virnsnap authors. See file "AUTHORS".
// Licensed under the MIT License. You have obtained a copy of the License at
// the "LICENSE" file in this repository.

// Package main implements the handlers for the different command line arguments.
package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/joroec/virsnap/pkg/virt"
	"github.com/spf13/cobra"
)

const (
	errNoVMsMatchingRegex = "no virtual machines found matching provided regex"
)

var (
	// keepVersions is a global variable determing how many successive snapshot
	// of the VM should be kept before beginning to remove the old ones.
	keepVersions int

	// assumeYes is a global variable determing whether snapshots should be deleted
	// without additional confirmation.
	assumeYes bool

	// cleanCmd is a global variable defining the corresponding cobra command
	cleanCmd = &cobra.Command{
		Use:   "clean [-y] -k <keep> <regex1> [<regex2>] [<regex3>] ...",
		Short: "Remove expired snapshots from the system",
		Long: "Remove expired snapshots from the system. The parameter k " +
			"specifies how many successive snapshots of a VM should be kept before " +
			"beginning to remove the old ones. For example, if you take a snapshot " +
			"every day over a time of 30 days and then call clean with an k of 15, " +
			"the oldest 15 snapshots that were created by libvirt get removed. The " +
			"regular expression(s) determine the VM names of those VM whose " +
			"snapshots should get cleaned. For example, 'virsnap clean -k 10 \".*\"' " +
			"cleans the snapshots of all found virtual machines, whereas " +
			"'virsnap clean -k 10 \"testing\"' cleans the snapshots only for those " +
			"virtial machines whose name includes \"testing\". ",
		Args: cobra.MinimumNArgs(1),
		Run:  cleanRun,
	}
)

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
		logger.Fatal("parameter k must not be negative")
	}

	vms, err := virt.ListMatchingVMs(logger, args, socketURL)
	if err != nil {
		logger.Fatalf("unable to retrieve virtual machines: %s", err)
	}

	defer virt.FreeVMs(logger, vms)

	if len(vms) == 0 {
		logger.Fatal(errNoVMsMatchingRegex)
	}
	logger.Debugf("found %d matching VMs", len(vms))

	if assumeYes {
		logger.Debugf("removing snapshots without any further confirmation")
	}

	// a boolean indicating whether at least one error occured. Useful for
	// the exit code of the program after iterating over the virtual machines.
	failed := false

	for _, vm := range vms {

		// iterate over the domains and clean the snapshots for each of it
		snapshots, err := vm.ListMatchingSnapshots([]string{".*"})
		if err != nil {
			logger.Errorf("skpping VM '%s': error, unable to get snapshot: %s",
				vm.Descriptor.Name,
				err,
			)
			failed = true
			continue
		}

		func() { // anonymous function for not calling snapshot.Free in a loop
			defer virt.FreeSnapshots(logger, snapshots)

			if len(snapshots) <= keepVersions {
				return
			}

			// iterate over the snapshot exceeding the k snapshots that should
			// remain
			for i := 0; i < len(snapshots)-keepVersions; i++ {
				logger.Infof("removing snapshot '%s' of VM '%s'.",
					snapshots[i].Descriptor.Name,
					vm.Descriptor.Name,
				)

				var accepted bool
				if assumeYes {
					accepted = true
				} else {
					accepted = confirm("Remove snapshot?", 10)
				}

				if accepted {
					logger.Infof("removing snapshot '%s' of VM '%s'.",
						snapshots[i].Descriptor.Name,
						vm.Descriptor.Name,
					)

					err = snapshots[i].Instance.Delete(0)
					if err != nil {
						logger.Errorf("skipping VM '%s': error, unable to remove snapshot '%s' of VM '%s': %s",
							vm.Descriptor.Name,
							snapshots[i].Descriptor.Name,
							err,
						)
						failed = true
						return // we are in an anonymous function
					}
				} else {
					logger.Infof("skipping removal of snapshot '%s' of VM '%s'",
						snapshots[i].Descriptor.Name,
						vm.Descriptor.Name,
					)
				}
			}
		}()
	}
	// TODO (obitech): improve error handling
	// See: https://blog.golang.org/errors-are-values
	if failed {
		logger.Fatal("clean process failed due to errors")
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
			logger.Fatal(err)
		}

		// Empty input (i.e. "\n")
		if len(res) < 2 {
			continue
		}

		return strings.ToLower(strings.TrimSpace(res))[0] == 'y'
	}

	return false
}
