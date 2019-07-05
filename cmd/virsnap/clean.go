// Copyright (c) 2019 Jonas R. <joroec@gmx.net>
// Licensed under the MIT License. You have obtained a copy of the License at
// the "LICENSE" file in this repository.

// Package cmd implements the handlers for the different command line arguments.
package cmd

import (
  "fmt"
  "strings"
  "sort"
  
  "github.com/spf13/cobra"
  "github.com/libvirt/libvirt-go"
  "github.com/libvirt/libvirt-go-xml"
  
  log "github.com/sirupsen/logrus"
  VM "github.com/joroec/virsnap/pkg/vm"
)

var keepVersions int

// cleanCmd is a global variable defining the corresponding cobra command
var cleanCmd = &cobra.Command{
  Use:   "clean -k <keep> <regex1> [<regex2>] [<regex3>] ...",
  Short: "Removes deprecated snapshots from the system.",
  Long:  `Removes deprecated snapshots from the system.`,
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

  // add command to root command so that cobra works as expected
  RootCmd.AddCommand(cleanCmd)
}

// DescribedSnapshot is a wrapper class for a snapshot Object together
// with its marshalled form from libvirtxml
type DescribedSnapshot struct {
  Obj libvirt.DomainSnapshot
  Desc libvirtxml.DomainSnapshot
}

func cleanRun(cmd *cobra.Command, args []string) {
  
  vms, err := VM.GetMatchingVMs(args)
  if err != nil {
    log.Fatal("Could not retrieve the virtual machines")
  }
  
  defer VM.FreeVMs(vms)
  
  for _, vm := range(vms) {
    
    // iterate over the domains and clean the snapshots for each of it
    func(){
      
      snapshots, err := vm.Domain.ListAllSnapshots(0)
      if err != nil {
        log.Error("Could not get the snapshot for VM:", vm.Name, err)
        return // we are in an anonymous function
      }
      
      // iterate over snapshot and check for prefix
      snapshot_Descs := make([]DescribedSnapshot, 0, len(snapshots))
      for _, snapshot := range(snapshots) {
        
        xml, err := snapshot.GetXMLDesc(0)
        if err != nil {
          log.Error("Could not get the snapshot xml for VM:", vm.Name,
            err)
          return // we are in an anonymous function
        }
        
        Descriptor := libvirtxml.DomainSnapshot{}
        err = Descriptor.Unmarshal(xml)
        if err != nil {
          log.Error("Could not unmarshal the snapshot xml for VM:", vm.Name,
            ". Skipping the VM.")
          return // we are in an anonymous function
        }
        
        if strings.HasPrefix(Descriptor.Name, "virsnap_") {
          Desc_snap := DescribedSnapshot{
            Obj: snapshot,
            Desc: Descriptor,
          }
          snapshot_Descs = append(snapshot_Descs, Desc_snap)
        } else {
          snapshot.Free()
        }
        
      }
      
      // TODO: insert quick check whether there are enough snapshots
      
      // sort the snapshots according to their creation date increasingly
      sorter := SnapshotSorter{
        Snapshots: &snapshot_Descs,
      }
      sort.Sort(&sorter)
      
      // remove snapshots that are to old and exceed the number of snapshots
      // to keep.
      
      // iterate over the snapshot exceeding the k snapshots that should
      // remain
      for i := 0; i < len(snapshot_Descs)-keepVersions; i++ {
        log.Info("Removing snapshot", snapshot_Descs[i].Desc.Name, "of VM", 
          vm.Name)
        err = snapshot_Descs[i].Obj.Delete(0)
        if err != nil {
          log.Error("Could not remove snapshot", snapshot_Descs[i].Desc.Name,
            "of VM", vm.Name)
        }
      }
      
      // TODO: Free Domain snapshots
      for _, item := range(snapshot_Descs) {
        fmt.Println("Item: ", item.Desc.Name, "Date:", item.Desc.CreationTime)
      }
      
      
      log.Trace("Leaving creation of snapshot for VM:", vm.Name)
    }()
    
  }
  
}

type SnapshotSorter struct {
  Snapshots *[]DescribedSnapshot
}

func (s *SnapshotSorter) Len() int {
  return len(*s.Snapshots)
}

func (s *SnapshotSorter) Less(i int, j int) bool {
  return (*s.Snapshots)[i].Desc.CreationTime < 
    (*s.Snapshots)[j].Desc.CreationTime
}

func (s *SnapshotSorter) Swap(i int, j int) {
  (*s.Snapshots)[i].Desc, (*s.Snapshots)[j].Desc = 
    (*s.Snapshots)[j].Desc, (*s.Snapshots)[i].Desc
}
