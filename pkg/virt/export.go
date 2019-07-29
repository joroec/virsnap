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

	libvirtxml "github.com/libvirt/libvirt-go-xml"
)

// Export is a function that exports a given VM.
func (vm *VM) Export(outputDirectory string, logger Logger) error {
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

	// TODO: TOCTOU race condition?

	// create the output directory for the VM if not already existing
	sanVMName := sanitize.BaseName(descriptor.Name)

	vmOutputDir := path.Join(outputDirectory, sanVMName)
	err = fs.EnsureDirectory(vmOutputDir)
	if err != nil {
		return err
	}

	// loop over HDDs and store them using differential file sync
	for _, disk := range descriptor.Devices.Disks {

		filepath := disk.Source.File.File
		if filepath == "" {
			err = fmt.Errorf("could not get filepath of disk %s", disk.Target)
			logger.Warnf("Skipping the disk: %s", err)
			continue
		}

		filename := path.Base(filepath)
		err = fs.Sync(filepath, path.Join(vmOutputDir, filename))
		if err != nil {
			err = fmt.Errorf("could sync the disk %s: %v", filepath, err)
			logger.Warnf("Skipping the disk: %s", err)
			continue
		}

		// transform descriptor
		// TODO: das geht auch nocht nicht! Irgendwie mit Pointern arbeiten!
		disk.Source.File.File = "./ssssssssss" + filename

	}

	// store new descriptor alongside the disk files
	xmldoc, err := descriptor.Marshal()
	if err != nil {
		err = fmt.Errorf("could marshal the new descriptor %v: %v", descriptor, err)
		return err
	}

	file, err := os.Create(path.Join(vmOutputDir, "descriptor.xml"))
	if err != nil {
		err = fmt.Errorf("could not open new descriptor file: %v", err)
		return err
	}
	defer file.Close()

	file.WriteString(xmldoc)

	return nil
}
