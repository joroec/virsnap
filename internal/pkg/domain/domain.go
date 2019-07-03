// Copyright (c) 2019 Jonas R. <joroec@gmx.net>
// Licensed under the MIT License. You have obtained a copy of the License at
// the "LICENSE" file in this repository.

// Package domain implements high-level functions for the virsnap tool that
// utilize from the more low-level libvirt functions.
package domain

import (
  "regexp"
  
  log "github.com/sirupsen/logrus"
  
  "github.com/libvirt/libvirt-go"
)

// Simple wrapper type for a libvirt.Domain with its name as string.
type DomWithName struct {
    Domain libvirt.Domain
    Name string
}

// GetMatchingDomains is a function that allows to retrieve information about
// domains that can be accessed via libvirt. The first parameter specifies a 
// slice of regular expressions. Only virtual machines whose name matches at
// least one of the regular expressions are returned. The caller is responsible
// for calling FreeDomains on the returned domains to free any buffer in
// libvirt.
func GetMatchingDomains(args []string) []DomWithName {
  log.Trace("Checking the validity of the given regular expressions...")
  regexes := make([]*regexp.Regexp, 0, len(args))
  for _, arg := range args {
    regex, err := regexp.Compile(arg)
    if err != nil {
      log.Error("Could not compile the regular expression ", arg)
      log.Fatal("Eror %s\n", err)
    }
    regexes = append(regexes, regex)
  }
  if len(regexes) == 0 {
    log.Panic("No (valid) regular expression was specified.")
  }
  log.Trace("Finished checking the validity of the regular expressions.")
  
  log.Trace("Trying to connect to QEMU socket...")
  conn, err := libvirt.NewConnect("qemu:///system")
  if err != nil {
    log.Error("Could not connect to QEMU socket. Exiting")
    log.Fatal("Eror %s\n", err)
  }
  defer conn.Close()
  log.Trace("Successfully connected to QEMU socket.")
  
  // The parameter for ListAllDomains is a bitmask that is used for filtering
  // the results. Since we do not want to restrict the usage to any strict type,
  // we use 0 which returns all of the found domains.
  log.Trace("Trying to retrieve all domains...")
  domains, err := conn.ListAllDomains(0)
  if err != nil {
    log.Error("Could not retrieve the QEMU domains. Exiting...")
    log.Fatal("Eror %s\n", err)
  }
  log.Trace("Successfully retrieved the QEMU domains.")
  
  // Loop over the domains and print out the name of the virtual machine.
  matched_domains := make([]DomWithName, 0, len(domains))
  for _, domain := range domains {
    
    log.Trace("Trying to retrieve a name for the domain...")
    name, err := domain.GetName()
    if err != nil {
      log.Error("Could not get the name of a domain. Skipping...")
      log.Error("Error: %s\n", err)
      continue
    }
    log.Trace("Successfully retrieved the name of a domain.")
    
    log.Trace("Trying to find a match to a regular expression.")

    // Only take the regexes into account if at least one has been specified.
    // Otherwise, assume any virtual machine is selected.
    if len(regexes) > 0 {
      for _, regex := range(regexes) {
        if regex.Find([]byte(name)) != nil {
          log.Trace("Found match for VM: ", name)
          matched_domains = append(matched_domains, DomWithName{Domain: domain,
            Name: name})
          break
        } else {
          // We do not need domain here anymore. The caller is responsible
          // for calling domain.Free() on the returned domains.
          domain.Free()
        }
      }
    } else {
      log.Trace("Found match for VM: ", name)
      matched_domains = append(matched_domains, DomWithName{Domain: domain,
        Name: name})
    }
  
  }
  
  return matched_domains
}

// FreeDomains is a function that takes a slice of DomWithName objects and
// frees any associated libvirt.Domain. Usually, this is called after
// GetMatchingDomains with a "defer" statement.
func FreeDomains(domains []DomWithName) {
  for _, domain := range(domains) {
    domain.Domain.Free()
  }
}
