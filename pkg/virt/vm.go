// Copyright (c) 2019 Jonas R. <joroec@gmx.net>
// Licensed under the MIT License. You have obtained a copy of the License at
// the "LICENSE" file in this repository.

// Package virt implements high-level functions for handling virtual machines
// (VMS) that use the more low-level libvirt functions internally.
package virt

import (
  "fmt"
  "regexp"
  "time"
  
  "github.com/libvirt/libvirt-go"
  "github.com/libvirt/libvirt-go-xml"
)

// -----------------------------------------------------------------------------

// VM is a simple wrapper type for a libvirt.Domain with its corresponding
// XML descriptor unmarshalled as data type.
type VM struct {
  Instance libvirt.Domain
  Descriptor libvirtxml.Domain
}

// TODO: add documentation
func (vm *VM) Free() error {
  return vm.Instance.Free()
}

// TODO: add documentation
// TODO: add parameter for force
func (vm *VM) Shutdown() (libvirt.DomainState, error) {
  former_state, _, err := vm.Instance.GetState()
  if err != nil {
    err = fmt.Errorf("Could not retrieve the state of the VM %s; "+
      "Error was: %v", vm.Descriptor.Name, err)
    return libvirt.DOMAIN_NOSTATE, err
  }
  
  Logger.Debug("Initial state of VM %s is %s", vm.Descriptor.Name, 
    GetStateString(former_state))
  
  switch former_state {

  case libvirt.DOMAIN_RUNNING:
    Logger.Debug("Sending shutdown request")
    err = vm.Instance.Shutdown() // returns instantly
    if err != nil {
      err = fmt.Errorf("Could not initiate the shutdown request for VM %s: %v",
        vm.Descriptor.Name, err)
      return libvirt.DOMAIN_RUNNING, err
    }
    
    // waiting for the VM to be in state shutoff
    // TODO: add a dynamic timeout later on
    new_state := libvirt.DOMAIN_RUNNING
    for i := 0; i < 15; i++ {
      new_state, _, err = vm.Instance.GetState()
      if err != nil {
        err = fmt.Errorf("Could not re-retrieve the state of the VM %s. "+
          "Trying again...: %v", vm.Descriptor.Name, err)
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
        vm.Descriptor.Name, GetStateString(new_state))
      return libvirt.DOMAIN_RUNNING, err
    }
    
    return libvirt.DOMAIN_RUNNING, nil
  
  case libvirt.DOMAIN_BLOCKED:
    err = fmt.Errorf("VM %s is blocked; dont know how to handle that!",
      vm.Descriptor.Name)
    return libvirt.DOMAIN_BLOCKED, err
    
  case libvirt.DOMAIN_PAUSED:
    err = fmt.Errorf("VM %s is paused; dont know how to handle that!",
      vm.Descriptor.Name)
    return libvirt.DOMAIN_PAUSED, err
  
  case libvirt.DOMAIN_SHUTDOWN:
    // since domain is being shut off, wait a little
    new_state := libvirt.DOMAIN_SHUTDOWN
    for i := 0; i < 15; i++ {
      new_state, _, err = vm.Instance.GetState()
      if err != nil {
        err = fmt.Errorf("Could not re-retrieve the state of the VM %s. "+
          "Trying again...: %v", vm.Descriptor.Name, err)
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
        vm.Descriptor.Name, GetStateString(new_state))
      return libvirt.DOMAIN_SHUTDOWN, err
    }
    
    return libvirt.DOMAIN_SHUTDOWN, nil
    
  case libvirt.DOMAIN_CRASHED:
    err = fmt.Errorf("VM %s has crashed; dont know how to handle that!",
      vm.Descriptor.Name)
    return libvirt.DOMAIN_CRASHED, err
  
  case libvirt.DOMAIN_PMSUSPENDED:
    err = fmt.Errorf("VM %s is pmsuspended; dont know how to handle that!",
      vm.Descriptor.Name)
    return libvirt.DOMAIN_PMSUSPENDED, err
  
  case libvirt.DOMAIN_SHUTOFF:
    return libvirt.DOMAIN_SHUTOFF, nil
    
  default:
    err = fmt.Errorf("Invalid VM %s state DOMAIN_NOSTATE", vm.Descriptor.Name)
    return libvirt.DOMAIN_NOSTATE, err
  }
}

// TODO: add documentation
func (vm *VM) Start() error {
  err := vm.Instance.Create()
  if err != nil {
    err = fmt.Errorf("Could not boot up the VM %s: %v", vm.Descriptor.Name, err)
    return err
  }
  return nil
}


// -----------------------------------------------------------------------------

// ListMatchingVMs is a function that allows to retrieve information about
// virtual machines that can be accessed via libvirt. The first parameter
// specifies a slice of regular expressions. Only virtual machines whose name
// matches at least one of the regular expressions are returned. The caller is
// responsible for calling FreeVMs on the returned slice to free any
// buffer in libvirt.
func ListMatchingVMs(regexes []string) ([]VM, error) {
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
  // we use 0 which returns all of the found virtual machines.
  instances, err := conn.ListAllDomains(0)
  if err != nil {
    err = fmt.Errorf("Could not get a list of virtual machines from QEMU: %v",
      err)
    return nil, err
  }
  
  // loop over the virtual machines and check for a match with the given
  // regular expressions
  matched_vms := make([]VM, 0, len(instances))
  for _, instance := range instances {
    
    // retrieve and unmarshal the descriptor of the VM
    xml, err := instance.GetXMLDesc(0)
    if err != nil {
      err = fmt.Errorf("Could not get the XML descriptor of a VM. "+
        "Skipping this VM: %v", err)
      Logger.Warning(err)
      continue
    }
    
    descriptor := libvirtxml.Domain{}
    err = descriptor.Unmarshal(xml)
    if err != nil {
      err = fmt.Errorf("Could not unmarshal the XML descriptor of a VM. "+
        "Skipping this VM: %v", err)
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
      matched_vm := VM{
        Instance: instance,
        Descriptor: descriptor,
      }
      matched_vms = append(matched_vms, matched_vm)
    } else {
      // we do not need the instance here anymore
      err = instance.Free()
      if err != nil {
        err = fmt.Errorf("Could not free the vm %s: %v", descriptor.Name, err)
        Logger.Warn(err)
      }
    }
  }
  
  return matched_vms, nil
}

// FreeVMs is a function that takes a slice of VMs and frees any associated 
// libvirt.Domain. Usually, this is called after ListMatchingVMs with a 
// "defer" statement.
func FreeVMs(vms []VM) {
  for _, vm := range(vms) {
    err := vm.Instance.Free()
    if err != nil {
      err = fmt.Errorf("Could not free the vm %s: %v", vm.Descriptor.Name, err)
      Logger.Warn(err)
    }
  }
}

// TODO: documentation
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
