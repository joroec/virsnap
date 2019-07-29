// Copyright (c) 2019 The virnsnap authors. See file "AUTHORS".
// Licensed under the MIT License. You have obtained a copy of the License at
// the "LICENSE" file in this repository.

// Package virt implements high-level functions for handling virtual machines
// (VMS) that use the more low-level libvirt functions internally.
package virt

import (
	"fmt"
	"os"
	"path"

	"github.com/joroec/virsnap/pkg/fs"
	"github.com/kennygrant/sanitize"

	"github.com/joroec/virsnap/pkg/instrument/log"

	libvirtxml "github.com/libvirt/libvirt-go-xml"
)

// Export is a function that exports a given VM.
func (vm *VM) Export(outputDirectory string, logger log.Logger) error {
	// get the XML descriptor
	xml, err := vm.Instance.GetXMLDesc(0)
	if err != nil {
		err = fmt.Errorf("unable to get XML descriptor of VM: %s", err)
		return err
	}

	descriptor := libvirtxml.Domain{}
	err = descriptor.Unmarshal(xml)
	if err != nil {
		err = fmt.Errorf("unable to unmarshal XML descriptor of VM: %s", err)
		return err
	}

	// create the output directory for the VM if not already existing
	sanVMName := sanitize.BaseName(vm.Descriptor.Name)

	vmOutputDir := path.Join(outputDirectory, sanVMName)
	err = fs.EnsureDirectory(vmOutputDir)
	if err != nil {
		return err
	}

	// loop over HDDs and store them using differential file sync
	for _, disk := range descriptor.Devices.Disks {
		// only observe disks, not cdroms
		if disk.Device != "disk" {
			continue
		}

		filepath := disk.Source.File.File
		if filepath == "" {
			logger.Errorf("could not get filepath of disk '%s'", disk.Target)
			continue
		}

		filename := path.Base(filepath)

		// transform descriptor
		disk.Source.File.File = "./" + filename

		// sync file
		err = fs.Sync(filepath, path.Join(vmOutputDir, filename), logger)
		if err != nil {
			logger.Errorf("could sync the disk '%s': %v", filepath, err)
		}
	}

	// store new descriptor alongside the disk files
	xmldoc, err := descriptor.Marshal()
	if err != nil {
		err = fmt.Errorf("could marshal the new descriptor '%v': %v", descriptor, err)
		return err
	}

	// create descriptor file if not existent, overwrite of existent
	file, err := os.Create(path.Join(vmOutputDir, "descriptor.xml"))
	if err != nil {
		err = fmt.Errorf("could not open new descriptor file: %v", err)
		return err
	}
	defer file.Close()

	file.WriteString(xmldoc)

	return nil
}
