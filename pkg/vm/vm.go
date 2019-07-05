// Copyright (c) 2019 Jonas R. <joroec@gmx.net>
// Licensed under the MIT License. You have obtained a copy of the License at
// the "LICENSE" file in this repository.

// Package vm implements high-level functions for handling virtual machines
// (VMS) that use the more low-level libvirt functions internally.
package vm

import (
  "fmt"
  "os"
  "regexp"
  "time"

  "github.com/sirupsen/logrus"
  "github.com/libvirt/libvirt-go"
)

// Log is the logrus object that is used of this package to print warning
// messages on the console. The default Logger is configured to use stdout for
// logging and sets a default level of Info. If you want to disable the logging
// output of this package, you need to set a member of the Logger from your
// own code:
// vm.Logger.Out = ioutil.Discard
// After this, no logging output is done anymore.
var Logger = logrus.New()

// init is a special golang function that is called exactly once regardless
// how often the package is imported.
func init() {
  Logger.Out = os.Stdout
  // Logger.SetLevel(logrus.WarnLevel)
  // TODO: uncomment
  Logger.SetLevel(logrus.TraceLevel)
}

// NamedVM is a simple wrapper type for a libvirt.Domain with its corresponding
// name as string.
type NamedVM struct {
    Domain libvirt.Domain
    Name string
}

// GetMatchingVMs is a function that allows to retrieve information about
// virtual machines that can be accessed via libvirt. The first parameter
// specifies a slice of regular expressions. Only virtual machines whose name
// matches at least one of the regular expressions are returned. The caller is
// responsible for calling FreeVMs on the returned domains to free any
// buffer in libvirt.
func GetMatchingVMs(regexes []string) ([]NamedVM, error) {
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
  
  // trying to connect to QEMU socket...
  conn, err := libvirt.NewConnect("qemu:///system")
  if err != nil {
    err = fmt.Errorf("Could not connect to QEMU socket: %v", err)
    return nil, err
  }
  defer conn.Close()
  
  // retrieving all virtual machines
  // the parameter for ListAllDomains is a bitmask that is used for filtering
  // the results. Since we do not want to restrict the usage to any strict type,
  // we use 0 which returns all of the found domains.
  domains, err := conn.ListAllDomains(0)
  if err != nil {
    err = fmt.Errorf("Could not get a list of virtual machines from QEMU: %v",
      err)
    return nil, err
  }
  
  // loop over the virtual machines and check for a match with the given
  // regular expressions
  matched_domains := make([]NamedVM, 0, len(domains))
  for _, domain := range domains {
    
    name, err := domain.GetName()
    if err != nil {
      err = fmt.Errorf("Could not get the name of a virtual machine. "+
        "Error was: %v", err)
      return nil, err
    }
    
    // checking for a matching regular expression
    found := false
    for _, regex := range(exprs) {
      if regex.Find([]byte(name)) != nil {
        found = true
        break
      }
    }
    
    if found {
      // the caller is responsible for calling domain.Free() on the returned
      // domains
      matched_domains = append(matched_domains, NamedVM{Domain: domain,
        Name: name})
    } else {
      // we do not need domain here anymore
      err = domain.Free()
      if err != nil {
        err = fmt.Errorf("Could not free the vm %s: %v", name, err)
        Logger.Warn(err)
      }
    }
  }
  
  return matched_domains, nil
}

// FreeDomains is a function that takes a slice of DomWithName objects and
// frees any associated libvirt.Domain. Usually, this is called after
// GetMatchingDomains with a "defer" statement.
func FreeVMs(vms []NamedVM) {
  for _, vm := range(vms) {
    err := vm.Domain.Free()
    if err != nil {
      err = fmt.Errorf("Could not free the vm %s: %v", vm.Name, err)
      Logger.Warn(err)
    }
  }
}


func GetStateString(state libvirt.DomainState) string {
  switch state {
  case libvirt.DOMAIN_RUNNING:
    return "DOMAIN_RUNNING"
  case libvirt.DOMAIN_BLOCKED:
    return "DOMAIN_BLOCKED"
  case libvirt.DOMAIN_PAUSED:
    return "DOMAIN_PAUSED"
  case libvirt.DOMAIN_SHUTDOWN:
    return "DOMAIN_SHUTDOWN"
  case libvirt.DOMAIN_CRASHED:
    return "DOMAIN_CRASHED"
  case libvirt.DOMAIN_PMSUSPENDED:
    return "DOMAIN_PMSUSPENDED"
  case libvirt.DOMAIN_SHUTOFF:
    return "DOMAIN_SHUTOFF"
  default:
    return "DOMAIN_NOSTATE"
  }
}


func Shutdown(vm NamedVM) (libvirt.DomainState, error) {
  former_state, _, err := vm.Domain.GetState()
  if err != nil {
    err = fmt.Errorf("Could not retrieve the state of the VM %s; "+
      "Error was: %v", vm.Name, err)
    return libvirt.DOMAIN_NOSTATE, err
  }
  Logger.Debug("Initial state of VM %s is %s", vm.Name, 
    GetStateString(former_state))
  
  switch former_state {

  case libvirt.DOMAIN_RUNNING:
    Logger.Debug("Sending shutdown request")
    err = vm.Domain.Shutdown() // returns instantly
    if err != nil {
      err = fmt.Errorf("Could not initiate the shutdown request for VM %s: %v",
        vm.Name, err)
      return libvirt.DOMAIN_RUNNING, err
    }
    
    // waiting for the VM to be in state shutoff
    // TODO: add a dynamic timeout later on
    new_state := libvirt.DOMAIN_RUNNING
    for i := 0; i < 15; i++ {
      new_state, _, err = vm.Domain.GetState()
      if err != nil {
        err = fmt.Errorf("Could not re-retrieve the state of the VM %s. "+
          "Trying again...: %v", vm.Name, err)
        Logger.Warn(err)
      }
      
      if new_state == libvirt.DOMAIN_SHUTOFF {
        break
      }
      
      time.Sleep(5 * time.Second)
    }
    
    // TODO: add a better error handling
    if new_state != libvirt.DOMAIN_SHUTOFF {
      err = fmt.Errorf("Could not shutdown the domain %s! State is %s!", 
        vm.Name, GetStateString(new_state))
      return libvirt.DOMAIN_RUNNING, err
    }
    
    return libvirt.DOMAIN_RUNNING, nil
  
  case libvirt.DOMAIN_BLOCKED:
    err = fmt.Errorf("VM is blocked; dont know how to handle that!")
    return libvirt.DOMAIN_BLOCKED, err
    
  case libvirt.DOMAIN_PAUSED:
    err = fmt.Errorf("VM is paused; dont know how to handle that!")
    return libvirt.DOMAIN_PAUSED, err
  
  case libvirt.DOMAIN_SHUTDOWN:
    // since domain is being shut off, wait a little
    new_state := libvirt.DOMAIN_SHUTDOWN
    for i := 0; i < 15; i++ {
      new_state, _, err = vm.Domain.GetState()
      if err != nil {
        err = fmt.Errorf("Could not re-retrieve the state of the VM %s. "+
          "Trying again...: %v", vm.Name, err)
        Logger.Warn(err)
      }
      
      if new_state == libvirt.DOMAIN_SHUTOFF {
        break
      }
      
      // TODO: exponential backoff, retry to send the shutdown request?
      time.Sleep(5 * time.Second)
    }
    
    // TODO: add a better error handling
    // TODO: refactor to an own function, since doubled code "waitStateChange"
    if new_state != libvirt.DOMAIN_SHUTOFF {
      err = fmt.Errorf("Could not shutdown the domain %s! State is %s!", 
        vm.Name, GetStateString(new_state))
      return libvirt.DOMAIN_SHUTDOWN, err
    }
    
    return libvirt.DOMAIN_SHUTDOWN, nil
    
    
  case libvirt.DOMAIN_CRASHED:
    err = fmt.Errorf("VM has crashed; dont know how to handle that!")
    return libvirt.DOMAIN_CRASHED, err
  
  case libvirt.DOMAIN_PMSUSPENDED:
    err = fmt.Errorf("VM is pmsuspended; dont know how to handle that!")
    return libvirt.DOMAIN_PMSUSPENDED, err
  
  case libvirt.DOMAIN_SHUTOFF:
    return libvirt.DOMAIN_SHUTOFF, nil
    
  default:
    err = fmt.Errorf("Invalid VM state DOMAIN_NOSTATE")
    return libvirt.DOMAIN_NOSTATE, err
  }
}


func Start(vm NamedVM) error {
  err := vm.Domain.Create()
  if err != nil {
    err = fmt.Errorf("Could not boot up the VM %s: %v", vm.Name, err)
    return err
  }
  
  return nil
}
