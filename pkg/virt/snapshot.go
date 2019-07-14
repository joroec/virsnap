// Copyright (c) 2019 The virnsnap authors. See file "AUTHORS".
// Licensed under the MIT License. You have obtained a copy of the License at
// the "LICENSE" file in this repository.

// Package virt implements high-level functions for handling virtual machines
// (VMS) that use the more low-level libvirt functions internally.
package virt

import (
	"fmt"
	"regexp"
	"sort"

	"github.com/docker/docker/pkg/namesgenerator"
	"github.com/libvirt/libvirt-go"
	libvirtxml "github.com/libvirt/libvirt-go-xml"
	"go.uber.org/zap"
)

// -----------------------------------------------------------------------------

// Snapshot is a simple wrapper type for a libvirt.DomainSnapshot with its
// corresponding XML descriptor unmarshalled as data type.
type Snapshot struct {
	Instance   libvirt.DomainSnapshot
	Descriptor libvirtxml.DomainSnapshot
}

// Free is a convenience method for calling Free on the corresponding libvirt
// Snapshot instance.
func (s *Snapshot) Free() error {
	return s.Instance.Free()
}

// -----------------------------------------------------------------------------

// ListMatchingSnapshots is a method that allows to retrieve information about
// virtual machine snapshots hat can be accessed via libvirt. The first
// parameter specifies a slice of regular expressions. Only snapshots of virtual
// machines whose name matches at least one of the regular expressions are
// returned. The caller is responsible for calling FreeSnapshots on the
// returned slice to free any buffer in libvirt. The returned snapshots
// are sorted by creation time.
func (vm *VM) ListMatchingSnapshots(regexes []string) ([]Snapshot, error) {
	// argument validity checking
	exprs := make([]*regexp.Regexp, 0, len(regexes))
	for _, arg := range regexes {
		regex, err := regexp.Compile(arg)
		if err != nil {
			err = fmt.Errorf("unable to compile regular expression %s: %v", arg,
				err)
			return nil, err
		}
		exprs = append(exprs, regex)
	}

	if len(exprs) == 0 {
		return nil, fmt.Errorf("no regular expression was specified")
	}

	// retrieve all snapshots from libvirt
	instances, err := vm.Instance.ListAllSnapshots(0)
	if err != nil {
		err = fmt.Errorf("unable to retrieve snapshots for VM %s: %v",
			vm.Descriptor.Name, err)
		return nil, err
	}

	matchedSnapshots := make([]Snapshot, 0, len(instances))

	// loop over snapshots and check for a match with the given
	// regular expressions
	for _, instance := range instances {

		// retrieve and unmarshal the descriptor of snapshot
		xml, err := instance.GetXMLDesc(0)
		if err != nil {
			err = fmt.Errorf("unable to get XML descriptor of snapshot: %s", err)
			vm.Logger.Warnf("Skipping snapshot: %s", err)
			continue
		}

		descriptor := libvirtxml.DomainSnapshot{}
		err = descriptor.Unmarshal(xml)
		if err != nil {
			err = fmt.Errorf("unable to unmarshal the XML descriptor of snapshot: %s", err)
			vm.Logger.Warn("Skipping snapshot: %v", err)
			continue
		}

		// checking for a matching regular expression
		found := false
		for _, regex := range exprs {
			if regex.Find([]byte(descriptor.Name)) != nil {
				found = true
				break
			}
		}

		if found {
			// the caller is responsible for calling domain.Free() on the returned
			// domains
			matchedSnapshot := Snapshot{
				Instance:   instance,
				Descriptor: descriptor,
			}
			matchedSnapshots = append(matchedSnapshots, matchedSnapshot)
		} else {
			// we do not need the instance here anymore
			err = instance.Free()
			if err != nil {
				vm.Logger.Warnf("unable to free snapshot %s: %v",
					descriptor.Name,
					err,
				)
			}
		}
	}

	// sort snapshots according to their creation date increasingly
	sorter := SnapshotSorter{
		Snapshots: &matchedSnapshots,
	}
	sort.Sort(&sorter)

	return matchedSnapshots, nil
}

// FreeSnapshots is a function that takes a slice of snapshots and frees any
// associated libvirt.DomainSnapshot. Usually, this is called after
// ListMatchingSnapshots with a "defer" statement.
func FreeSnapshots(log *zap.SugaredLogger, snapshots []Snapshot) {
	for _, snapshot := range snapshots {
		err := snapshot.Instance.Free()
		if err != nil {
			log.Warnf("unable to free snapshot %s: %s",
				snapshot.Descriptor.Name,
				err,
			)
		}
	}
}

// CreateSnapshot creates a snapshot for the given domain while checking
// whether the name is already used. The given prefix is prepended to the
// snapshots name. The caller is responsible for calling Free on snapshot.
func (vm *VM) CreateSnapshot(prefix string, description string) (Snapshot,
	error) {
	var descriptor libvirtxml.DomainSnapshot

	for true {
		descriptor = libvirtxml.DomainSnapshot{
			Name:        prefix + namesgenerator.GetRandomName(0),
			Description: description,
		}

		// check if name is already given
		regex := []string{"^" + descriptor.Name + "$"}
		snapshots, err := vm.ListMatchingSnapshots(regex)
		if err != nil {
			err = fmt.Errorf("unable to retrieve existing snapshot for VM '%s': %v",
				vm.Descriptor.Name,
				err,
			)
			return Snapshot{}, err
		}

		if len(snapshots) == 0 {
			break
		}
	}

	// create snapshot with the given name
	xml, err := descriptor.Marshal()
	if err != nil {
		err = fmt.Errorf("unable to marshal snapshot XML for VM '%s': %s",
			vm.Descriptor.Name,
			err,
		)
		return Snapshot{}, err
	}

	snapshot, err := vm.Instance.CreateSnapshotXML(xml, 0)
	if err != nil {
		err = fmt.Errorf("unable to create snapshot for VM '%s': %s",
			vm.Descriptor.Name,
			err,
		)
		return Snapshot{}, err
	}

	return Snapshot{
		Instance:   *snapshot,
		Descriptor: descriptor,
	}, nil
}

// -----------------------------------------------------------------------------

// SnapshotSorter is a sorter for sorting snapshots by creation date.
type SnapshotSorter struct {
	Snapshots *[]Snapshot
}

func (s *SnapshotSorter) Len() int {
	return len(*s.Snapshots)
}

func (s *SnapshotSorter) Less(i int, j int) bool {
	return (*s.Snapshots)[i].Descriptor.CreationTime <
		(*s.Snapshots)[j].Descriptor.CreationTime
}

func (s *SnapshotSorter) Swap(i int, j int) {
	(*s.Snapshots)[i], (*s.Snapshots)[j] =
		(*s.Snapshots)[j], (*s.Snapshots)[i]
}
