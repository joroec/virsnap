// Copyright (c) 2019 Jonas R. <joroec@gmx.net>
// Licensed under the MIT License. You have obtained a copy of the License at
// the "LICENSE" file in this repository.

// Package cmd implements the handlers for the different command line arguments.
package cmd

import (
  "fmt"

  "github.com/spf13/cobra"
  log "github.com/sirupsen/logrus"
  
  "github.com/joroec/virsnap/internal/pkg/domain"
)

// listvmsCmd is a global variable defining the corresponding cobra command
var listvmsCmd = &cobra.Command{
  Use:   "listvms [<regex1> [<regex2> ...]]",
  Short: "List the virtual machines that can be detected via using libvirt.",
  Long:  "List the virtual machines that can be detected via using libvirt. "+
    "This is meant to be a simple method of testing both your connection to "+
    "the libvirt daemon and regular expressions for virtual machine "+
    "selection. For example, 'virsnap listvms \".*\"' prints all accessible "+
    "virtual machines, whereas 'virsnap listvms \"testing\"' prints only "+
    "virtual machines whose name includes \"testing\". If no regex is given, "+
    "any acccessible virtual machine is printed.",
  Run: listvmsRun,
}

// init is a special golang function that is called exactly once regardless
// how often the package is imported.
func init() {
  // add command to root command so that cobra works as expected
  RootCmd.AddCommand(listvmsCmd)
}

// listvmsRun is the function called after the command line parser detected
// that we want to end up here. This functions connects to the libvirt daemon,
// retrieves the current list of virtual machines and prints it to standard
// output.
func listvmsRun(cmd *cobra.Command, args []string) {
  log.Trace("Start execution of listvmsRun function.")
  
  var domains []domain.DomWithName
  if len(args) > 0 {
    // a regex has been specified, so we take it to filter the virtual machines
    domains = domain.GetMatchingDomains(args)
  } else {
    // listvms should display any virtual machine found. So, we need to specify
    // a search regex that matches any virtual machine name.
    regex := []string{".*"}
    domains = domain.GetMatchingDomains(regex)
  }
  defer domain.FreeDomains(domains)
  
  for _, domain := range(domains) {
    fmt.Println(domain.Name)
  }
  
  log.Trace("Returning from listvmsRun function.")
}