// Copyright (c) 2019 The virnsnap authors. See file "AUTHORS".
// Licensed under the MIT License. You have obtained a copy of the License at
// the "LICENSE" file in this repository.

// Package main implements the handlers for the different command line arguments.
package main

import (
	"github.com/joroec/virsnap/pkg/virt"
	"github.com/libvirt/libvirt-go"
	"github.com/spf13/cobra"
)

var (
	// shutdown is a global variable determing whether virsnap should try to
	// shutdown the virtual machine before taking the snapshot
	shutdown bool

	// force is a global variable determing whether virsnap should force the
	// shutdown of virtual machine before taking the snapshot
	force bool

	// timeout is a global variable determing the timeout in minutes to wait for a
	// graceful shutdown before forcing the shutdown if enabled or returning with
	// an error code
	timeout int

	// createCmd is a global variable defining the corresponding cobra command
	createCmd = &cobra.Command{
		Use:   "create <regex1> [<regex2>] [<regex3>] ...",
		Short: "Create a snapshot of one or more virtual machines",
		Long: "Create a new snapshot for any found virtual machine with a name " +
			"matching at least one of the given regular expressions. For example, " +
			"'virsnap create \".*\"' creates a new snapshot for all found virtual " +
			"machines, whereas 'virsnap create \"testing\"' creates a new snapshot " +
			"only for those virtial machines whose name includes \"testing\". The " +
			"snapshot will be assigned a random name. In any case, the name starts " +
			"with the prefix 'virsnap_'. virsnap expects the virtual machines " +
			"configured according to the personal snapshot preferences. If you want " +
			"to use QCOW2 internal snapshots, for example, edit the VM's XML " +
			"descriptor ('virsh edit <vm_name>') of the VM so that the default " +
			"snapshot behavior uses internal snapshots. Example: \n" +
			`
<disk type='file' device='disk' snapshot='internal'>
  <driver name='qemu' type='qcow2'/>
  <source file='/.../testing.qcow2'/>
  <backingStore/>
  <target dev='hda' bus='ide'/>
  <alias name='ide0-0-0'/>
  <address type='drive' controller='0' bus='0' target='0' unit='0'/>
</disk>`,
		Args: cobra.MinimumNArgs(1),
		Run:  createRun,
	}
)

// init is a special golang function that is called exactly once regardless
// how often the package is imported.
func init() {
	// initialize flags and arguments needed for this command
	createCmd.Flags().BoolVarP(&shutdown, "shutdown", "s", false, "Try to "+
		"shutdown the VM before making the snapshot. Restores state afterwards.")

	createCmd.Flags().BoolVarP(&force, "force", "f", false, "Force the "+
		"shutdown of the virtual machine. This flag can be combined with -s "+
		"exclusively.")

	createCmd.Flags().IntVarP(&timeout, "timeout", "t", 3, "Timeout in minutes "+
		"to wait for a virtual machine to shutdown gracefully before returning an "+
		"error code or forcing the shutdown (flag -f). This flag is only "+
		"combinable with -s and -f . If the timeout expires and force is "+
		"specified, plug the power cord to bring the machine down.")

	// add command to root command so that cobra works as expected
	RootCmd.AddCommand(createCmd)
}

// createRun takes as parameter the regular expressions of the names of the VMs
// to create a snapshot for
func createRun(cmd *cobra.Command, args []string) {
	// check the validity of the console line parameters
	if force && !shutdown {
		logger.Fatal("flag -f can only be specified if -s was specified!")
	}

	if timeout <= 0 {
		logger.Fatal("nvalid timeout specified. Must be greater than zero!")
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
	failed := false

	for _, vm := range vms {
		// iterate over the domains and crete a new snapshot for each of it
		formerState := libvirt.DOMAIN_NOSTATE
		if shutdown {
			formerState, err = vm.Transition(libvirt.DOMAIN_SHUTOFF, force, timeout)
			if err != nil {
				logger.Error(err)
				failed = true
				continue
			}
		}

		logger.Debugf("Beginning creation of snapshot for VM '%s'.",
			vm.Descriptor.Name,
		)

		snapshot, err := vm.CreateSnapshot("virsnap_",
			"snapshot created by virnsnap")
		if err == nil {
			logger.Infof("Created snapshot '%s' for VM '%s'",
				snapshot.Descriptor.Name, vm.Descriptor.Name)
		} else {
			logger.Errorf("unable to create snapshot for VM: '%s': %s",
				vm.Descriptor.Name,
				err,
			)
			failed = true
			// no continue here, since we want to startup the VM is any case!
		}

		func() { // anonymous function for not calling snapshot.Free in a loop
			defer snapshot.Free()

			if shutdown {
				logger.Debugf("Restoring previous state of vm '%s'",
					vm.Descriptor.Name,
				)
				_, err = vm.Transition(formerState, force, timeout)
				if err != nil {
					logger.Errorf("unable to restore state '%s' of VM '%s': %s",
						virt.GetStateString(formerState),
						vm.Descriptor.Name,
						err,
					)
					failed = true

					newState, err := vm.GetCurrentStateString()
					if err != nil {
						logger.Errorf("unable to retrieve current state of VM ;;'%s': %s ",
							vm.Descriptor.Name,
							err,
						)
						return // we are in an anonymous function
					}

					logger.Warnf("state of VM '%s' is now '%s'", vm.Descriptor.Name,
						newState)
					return // we are in an anonymous function
				}
			}

			logger.Debugf("Finished creation of snapshot '%s' for VM '%s'.",
				snapshot.Descriptor.Name,
				vm.Descriptor.Name,
			)
		}()

	}

	// TODO (obitech): improve error handling
	// See: https://blog.golang.org/errors-are-values
	if failed {
		logger.Fatal("create process failed due to errors")
	}

}
