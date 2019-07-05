// Copyright (c) 2019 Jonas R. <joroec@gmx.net>
// Licensed under the MIT License. You have obtained a copy of the License at
// the "LICENSE" file in this repository.

// Package virt implements high-level functions for handling virtual machines
// (VMS) that use the more low-level libvirt functions internally.
package virt

import (
  "fmt"
  "regexp"
  "sort"
  
  "github.com/libvirt/libvirt-go"
  "github.com/libvirt/libvirt-go-xml"
)

// -----------------------------------------------------------------------------

// Snapshot is a simple wrapper type for a libvirt.DomainSnapshot with its
// corresponding XML descriptor unmarshalled as data type.
type Snapshot struct {
  Instance libvirt.DomainSnapshot
  Descriptor libvirtxml.DomainSnapshot
}

// TODO: add documentation
func (s *Snapshot) Free() error {
  return s.Instance.Free()
}

// -----------------------------------------------------------------------------

// TODO: add documentation
// returns sorted increasingly accoriding to SnapshotCreation data
func (vm *VM) ListMatchingSnapshots(regexes []string) ([]Snapshot, error) {
  // argument validity checking
  exprs := make([]*regexp.Regexp, 0, len(regexes))
  for _, arg := range regexes {
    regex, err := regexp.Compile(arg)
    if err != nil {
      err = fmt.Errorf("Could not compile the regular expression %s: %v", arg, 
        err)
      return nil, err
    }
    exprs = append(exprs, regex)
  }
  
  if len(exprs) == 0 {
    return nil, fmt.Errorf("No regular expression was specified.")
  }
  
  // retrieve all snapshots from libvirt
  instances, err := vm.Instance.ListAllSnapshots(0)
  if err != nil {
    err = fmt.Errorf("Could not retrieve the snapshots for the VM %s: %v", 
      vm.Descriptor.Name, err)
    return nil, err
  }
  
  matched_snapshots := make([]Snapshot, 0, len(instances))
  
  // loop over the snapshots and check for a match with the given
  // regular expressions
  for _, instance := range(instances) {
    
    // retrieve and unmarshal the descriptor of the snapshot
    xml, err := instance.GetXMLDesc(0)
    if err != nil {
      err = fmt.Errorf("Could not get the XML descriptor of a snapshot. "+
        "Skipping this snapshot: %v", err)
      Logger.Warning(err)
      continue
    }
    
    descriptor := libvirtxml.DomainSnapshot{}
    err = descriptor.Unmarshal(xml)
    if err != nil {
      err = fmt.Errorf("Could not unmarshal the XML descriptor of a snapshot. "+
        "Skipping this snapshot: %v", err)
      Logger.Warning(err)
      continue
    }
    
    // checking for a matching regular expression
    found := false
    for _, regex := range(exprs) {
      if regex.Find([]byte(descriptor.Name)) != nil {
        found = true
        break
      }
    }
    
    if found {
      // the caller is responsible for calling domain.Free() on the returned
      // domains
      matched_snapshot := Snapshot{
        Instance: instance,
        Descriptor: descriptor,
      }
      matched_snapshots = append(matched_snapshots, matched_snapshot)
    } else {
      // we do not need the instance here anymore
      err = instance.Free()
      if err != nil {
        err = fmt.Errorf("Could not free the snapshot %s: %v", descriptor.Name,
          err)
        Logger.Warn(err)
      }
    }
  }
  
  // sort the snapshots according to their creation date increasingly
  sorter := SnapshotSorter{
    Snapshots: &matched_snapshots,
  }
  sort.Sort(&sorter)
  
  return matched_snapshots, nil
}

// FreeSnapshots is a function that takes a slice of snapshots and frees any
// associated libvirt.DomainSnapshot. Usually, this is called after 
// ListMatchingSnapshots with a "defer" statement.
func FreeSnapshots(snapshots []Snapshot) {
  for _, snapshot := range(snapshots) {
    err := snapshot.Instance.Free()
    if err != nil {
      err = fmt.Errorf("Could not free the snapshot %s: %v", 
        snapshot.Descriptor.Name, err)
      Logger.Warn(err)
    }
  }
}

// -----------------------------------------------------------------------------

// TODO: documentation
type SnapshotSorter struct {
  Snapshots *[]Snapshot
}

// TODO: documentation
func (s *SnapshotSorter) Len() int {
  return len(*s.Snapshots)
}

// TODO: documentation
func (s *SnapshotSorter) Less(i int, j int) bool {
  return (*s.Snapshots)[i].Descriptor.CreationTime < 
    (*s.Snapshots)[j].Descriptor.CreationTime
}

// TODO: documentation
func (s *SnapshotSorter) Swap(i int, j int) {
  (*s.Snapshots)[i], (*s.Snapshots)[j] = 
    (*s.Snapshots)[j], (*s.Snapshots)[i]
}
