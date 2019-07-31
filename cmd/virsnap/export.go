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
)

const (
	// filemode denotes the access rights for created files and directories.
	filemode = 0700
)

var (
	// outputDir is the target directory of the backup
	outputDir string

	// snapshotAfterShutdown determines whether virsnap should make a new
	// snapshot after the machine was shut down.
	snapshotAfterShutdown = true

	// exportCmd is a global variable defining the corresponding cobra command
	exportCmd = &cobra.Command{
		Use:   "export --output-dir <export_directory> <regex1> [<regex2>] [<regex3>] ...",
		Short: "Export a VM by copying the hard drive images to an output directory",
		Long: "Export a VM by copying the hard drive images and an copy of the " +
			"VMs XML descriptor file to an output directory. Exports any found " +
			"virtual machine with a name matching at least one of the given " +
			"regular expressions. To ensure a safe export, the VM needs to be " +
			"shutoff. Hence, virsnap shuts down the VM if its running, exports the " +
			"disk files and restores the VM's previous state afterwards. Apart from " +
			"this, there is an option to create a snapshot of the VM after " +
			"shutdowning and before exporting to the given directory.",
		Args: cobra.MinimumNArgs(1),
		Run:  exportRun,
	}
)

// init is a special golang function that is called exactly once regardless
// how often the package is imported.
func init() {
	// initialize flags and arguments needed for this command
	exportCmd.Flags().StringVarP(&outputDir, "output-dir", "o", "default?",
		"desc")
	exportCmd.MarkFlagRequired("output-dir")

	exportCmd.Flags().BoolVarP(&snapshotAfterShutdown, "snapshot", "s", true,
		"Create a new snapshot after the machine has been shut down.")

	exportCmd.Flags().IntVarP(&timeout, "timeout", "t", 3, "Timeout in minutes "+
		"to wait for a virtual machine to shutdown gracefully before forcing the "+
		"shutdown (flag -f). If the timeout expires and force is specified, plug "+
		"the power cord to bring the machine down.")

	// add command to root command so that cobra works as expected
	RootCmd.AddCommand(exportCmd)
}

// exportRun takes as parameter the regular expressions of the names of the VMs
// to export to the given output directory
func exportRun(cmd *cobra.Command, args []string) {
	// check the validity of the console line parameters
	absOutputDir, err := filepath.Abs(outputDir)
	if err != nil {
		logger.Fatalf("could not parse outputDir filepath '%s': %v", outputDir, err)
	}

	err = os.MkdirAll(absOutputDir, filemode)
	if err != nil {
		logger.Fatalf("could not create the output directory: %s", err)
	}

	vms, err := virt.ListMatchingVMs(logger, args, socketURL)
	if err != nil {
		logger.Fatal("could not retrieve virtual machines.")
	}
	defer virt.FreeVMs(logger, vms)

	if len(vms) == 0 {
		logger.Fatal(errNoVMsMatchingRegex)
	}

	// a boolean indicating whether at least one error occured. Useful for
	// the exit code of the program after iterating over the virtual machines.
	var failed bool

	// iterate over the VMs, shut them down and export them
	for _, vm := range vms {

		logger.Debugf("starting to shutdown VM '%s'", vm.Descriptor.Name)
		formerState, err := vm.Transition(libvirt.DOMAIN_SHUTOFF, true, timeout)
		if err != nil {
			logger.Error(err)
			failed = true
			continue
		}
		logger.Debugf("finshed shutdown process of VM '%s'", vm.Descriptor.Name)

		// scoped block for efficiently restoring the previous state of the VM
		{
			// restore previous state of VM
			defer func() {
				logger.Debugf("restoring previous state of vm '%s'", vm.Descriptor.Name)

				_, err = vm.Transition(formerState, true, timeout)
				if err != nil {
					logger.Errorf("unable to restore state '%s' of VM '%s': %s",
						virt.GetStateString(formerState), vm.Descriptor.Name, err)
					failed = true

					newState, err := vm.GetCurrentStateString()
					if err != nil {
						logger.Errorf("unable to retrieve current state of VM '%s': %s ",
							vm.Descriptor.Name, err)
					}

					logger.Warnf("state of VM '%s' is now '%s'", vm.Descriptor.Name,
						newState)
				}
			}()

			// should we create a snapshot after the VM has been shutdown?
			if snapshotAfterShutdown {
				logger.Debugf("Beginning creation of snapshot for VM '%s'.",
					vm.Descriptor.Name)

				snap, err := vm.CreateSnapshot("virsnap_", "snapshot created by virnsnap")
				if err == nil {
					logger.Infof("Created snapshot '%s' for VM '%s'", snap.Descriptor.Name,
						vm.Descriptor.Name)
				} else {
					logger.Errorf("unable to create a snapshot for the VM '%s': %s ",
						vm.Descriptor.Name, err)
					logger.Errorf("exporting VM '%s' without new snapshot", vm.Descriptor.Name)
					failed = true
				}
				snap.Free()
			}

			// do the actual export job, whenever we exit the scope of the
			// scoped block, we restore the previous state of the VM
			logger.Debugf("starting export process of VM '%s'", vm.Descriptor.Name)
			err = vm.Export(absOutputDir, filemode, logger)
			if err != nil {
				logger.Errorf("could not export the VM '%s': %v", vm.Descriptor.Name, err)
				failed = true
			}
			logger.Infof("Exported VM '%s'", vm.Descriptor.Name)

		}
	}

	// TODO (obitech): improve error handling
	// See: https://blog.golang.org/errors-are-values
	if failed {
		logger.Fatal("export process failed due to errors")
	}
}
