// Copyright (c) 2019 The virnsnap authors. See file "AUTHORS".
// Licensed under the MIT License. You have obtained a copy of the License at
// the "LICENSE" file in this repository.

// Package main implements the handlers for the different command line arguments.
package main

import (
	"os"
	"path/filepath"

	"github.com/joroec/virsnap/pkg/virt"
	"github.com/libvirt/libvirt-go"
	"github.com/spf13/cobra"
	"golang.org/x/sys/unix"
)

var (
	// outputDir is the target directory of the backup
	outputDir = ""

	// snapshot determines whether virsnap should make a new snapshot after the
	// machine was shut down.
	snapshot = true

	// exportCmd is a global variable defining the corresponding cobra command
	exportCmd = &cobra.Command{
		Use:   "export --output-dir <regex1> [<regex2>] [<regex3>] ...",
		Short: "Copy the hard drive images of a VM to an export directory",
		Long:  "TODO: write",
		Args:  cobra.MinimumNArgs(1),
		Run:   exportRun,
	}
)

// init is a special golang function that is called exactly once regardless
// how often the package is imported.
func init() {
	// initialize flags and arguments needed for this command
	exportCmd.Flags().StringVarP(&outputDir, "output-dir", "o", "default?",
		"desc")
	exportCmd.MarkFlagRequired("output-dir")

	exportCmd.Flags().BoolVarP(&snapshot, "snapshot", "s", true, "Make new "+
		"snapshot after shutdowning the machine.")

	exportCmd.Flags().IntVarP(&timeout, "timeout", "t", 3, "Timeout in minutes "+
		"to wait for a virtual machine to shutdown gracefully before forcing the "+
		"shutdown (flag -f). If the timeout expires and force is specified, plug "+
		"the power cord to bring the machine down.")

	// add command to root command so that cobra works as expected
	RootCmd.AddCommand(exportCmd)
}

// createRun takes as parameter the name of the VMs to create a snapshot for
func exportRun(cmd *cobra.Command, args []string) {
	// check the validity of the console line parameters
	absOutputDir, err := filepath.Abs(outputDir)
	if err != nil {
		logger.Fatalf("could not parse outputDir filepath: %v", err)
	}

	stat, err := os.Stat(absOutputDir)
	if err == nil {
		if os.IsNotExist(err) {
			// if path does not already exist: try to create target directory.
			err = os.Mkdir(absOutputDir, 0700)
			if err != nil {
				logger.Fatalf("could not create output directory: %v", err)
			}
		} else {
			// another error occured
			logger.Fatalf("could not parse outputDir filepath: %v", err)
		}
	}

	// check if stat denotes a directory
	if !stat.IsDir() {
		logger.Fatal("output directory does not point toa directory")
	}

	// check permissions for this directory
	err = unix.Access(absOutputDir, unix.R_OK|unix.W_OK)
	if err != nil {
		logger.Fatal("output directory is not writable or readable")
	}

	vms, err := virt.ListMatchingVMs(logger, args)
	if err != nil {
		logger.Fatal("could not retrieve virtual machines.")
	}
	defer virt.FreeVMs(logger, vms)

	if len(vms) == 0 {
		logger.Fatal(errNoVMsMatchingRegex)
	}

	// a boolean indicating whether at least one error occured. Useful for
	// the exit code of the program after iterating over the virtual machines.
	failed := false

	for _, vm := range vms {
		// iterate over the VMs, shut them down and export them
		formerState, err := vm.Transition(libvirt.DOMAIN_SHUTOFF, true, timeout)
		if err != nil {
			logger.Error(err)
			failed = true
			continue
		}

		// should we create a snapshot after the VM has been shutdown?
		if snapshot {
			logger.Debugf("Beginning creation of snapshot for VM '%s'.",
				vm.Descriptor.Name,
			)

			snap, err := vm.CreateSnapshot("virsnap_",
				"snapshot created by virnsnap")
			if err == nil {
				logger.Infof("Created snapshot '%s' for VM '%s'",
					snap.Descriptor.Name, vm.Descriptor.Name)
			} else {
				logger.Errorf("unable to create snapshot for VM: '%s': %s",
					vm.Descriptor.Name,
					err,
				)
				failed = true
				// no continue here, since we want to startup the VM is any case!
			}
			snap.Free() // we dont need to snapshot object any longer
		}

		// export the VM
		// TODO: implement

		// restore previous state
		logger.Debugf("restoring previous state of vm '%s'", vm.Descriptor.Name)

		_, err = vm.Transition(formerState, true, timeout)
		if err != nil {
			logger.Errorf("unable to restore state '%s' of VM '%s': %s",
				virt.GetStateString(formerState),
				vm.Descriptor.Name,
				err,
			)
			failed = true

			newState, err := vm.GetCurrentStateString()
			if err != nil {
				logger.Errorf("unable to retrieve current state of VM '%s': %s ",
					vm.Descriptor.Name,
					err,
				)
				continue
			}

			logger.Warnf("state of VM '%s' is now '%s'", vm.Descriptor.Name,
				newState)
			continue
		}

		logger.Debugf("Finished export of VM '%s'.", vm.Descriptor.Name)
	}

	// TODO (obitech): improve error handling
	// See: https://blog.golang.org/errors-are-values
	if failed {
		logger.Fatal("export process failed due to errors")
	}
}
