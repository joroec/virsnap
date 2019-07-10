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
  "strings"
  
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

// Free ist just a convenience function to free the associated libvirt.Domain
// instance.
func (vm *VM) Free() error {
  return vm.Instance.Free()
}

func (vm *VM) Transition(to libvirt.DomainState, forceShutdown bool, 
    timeout int) (libvirt.DomainState, error)  {
  
  // check argument validity
  if to != libvirt.DOMAIN_RUNNING && to != libvirt.DOMAIN_SHUTOFF &&
      to != libvirt.DOMAIN_PMSUSPENDED && to != libvirt.DOMAIN_PAUSED {
    err := fmt.Errorf("Cannot start the transition of VM \"%s\" to state "+
      "\"%s\". Target state is not allowed!", vm.Descriptor.Name,
      GetStateString(to))
    return libvirt.DOMAIN_NOSTATE, err
  }
  
  // get current state of virtual machine
  state, _, err := vm.Instance.GetState()
  if err != nil {
    err = fmt.Errorf("Could not retrieve the state of the VM \"%s\"; "+
      "Error was: %v", vm.Descriptor.Name, err)
    return libvirt.DOMAIN_NOSTATE, err
  }
  
  // outer switch: current state of VM, inner switch: target state of VM
  switch state {
  case libvirt.DOMAIN_RUNNING:
    
    switch to {
    case libvirt.DOMAIN_RUNNING:
      Logger.Debugf("Domain \"%s\" is already running.", vm.Descriptor.Name)
      return state, nil
    
    case libvirt.DOMAIN_PAUSED:
      Logger.Debugf("Suspending domain \"%s\".", vm.Descriptor.Name)
      err = vm.Instance.Suspend()
      if err != nil {
        err = fmt.Errorf("Could not suspend the VM \"%s\"; Error was: %v",
          vm.Descriptor.Name, err)
        return state, err
      }
      return state, nil
  
    case libvirt.DOMAIN_PMSUSPENDED:
      Logger.Debugf("PMSuspending domain \"%s\".", vm.Descriptor.Name)
      err = vm.Instance.PMSuspendForDuration(libvirt.NODE_SUSPEND_TARGET_MEM,
        0, 0)
      if err != nil {
        err = fmt.Errorf("Could not pmsuspend the VM \"%s\"; Error was: %v",
          vm.Descriptor.Name, err)
        return state, err
      }
      return state, nil
      
    case libvirt.DOMAIN_SHUTOFF:
      Logger.Debugf("Trying to shutdown domain \"%s\" gracefully.",
        vm.Descriptor.Name)
      
      round_seconds := 0.33 * float64(timeout*60)
      new_state := libvirt.DOMAIN_RUNNING

      // if the virtual machine seems to not react to the first shutdown
      // request, repeatedly send further requests to gracefully shutdown
      for i := 0; i < 3; i++ {
        before := time.Now()
        
        Logger.Debugf("Sending shutdown request to VM \"%s\".",
          vm.Descriptor.Name)
        err = vm.Instance.Shutdown() // returns instantly
        if err != nil {
          // we need to cast to specific libvirt error, since the VM might
          // be in a shutoff state since last check. If this is the case, we
          // do not want to return an error!
          lverr, ok := err.(libvirt.Error)
          if ok && (lverr.Code == libvirt.ERR_OPERATION_INVALID ||
              strings.Contains(lverr.Message, "domain is not running")) {
            Logger.Debugf("VM \"%s\" was shutdown in the meantime.",
              vm.Descriptor.Name)
            return libvirt.DOMAIN_RUNNING, nil 
          
          } else {
            err = fmt.Errorf("Could not initiate the shutdown request for "+
              "VM \"%s\": %v", vm.Descriptor.Name, err)
            return libvirt.DOMAIN_RUNNING, err
          }
        }
        
        Logger.Debugf("Waiting vor the VM \"%s\" to shutdown.", 
          vm.Descriptor.Name)
        for true {
          
          // sleep some seconds
          time.Sleep(5 * time.Second)
          
          new_state, _, err = vm.Instance.GetState()
          if err != nil {
            err = fmt.Errorf("Could not re-retrieve the state of the VM "+
              "\"%s\". Trying again...: %v", vm.Descriptor.Name, err)
            Logger.Warn(err)
          }
          
          if new_state == libvirt.DOMAIN_SHUTOFF {
            return libvirt.DOMAIN_RUNNING, nil
          }
          
          // if we waited longer since 33% of the timeout, try sending the
          // shutdown request again
          after := time.Now()
          duration := after.Sub(before) // int64 nanosecods
          max_round_duration := time.Duration(round_seconds)*time.Second
          if duration > max_round_duration {
            Logger.Debugf("Beginning next graceful shutdown round for VM "+
              "\"%s\".", vm.Descriptor.Name)
            break
          }
        }  
      }
      
      // could not shutdown the VM gracefully, force?
      if forceShutdown {
        Logger.Debugf("Destroying the VM \"%s\" since it could not be "+
          "shutdown gracefully.", vm.Descriptor.Name)
        err = vm.Instance.Destroy()
        if err != nil {
          err = fmt.Errorf("Could not destroy the VM \"%s\": %v",
            vm.Descriptor.Name, err)
          return libvirt.DOMAIN_RUNNING, err
        }
        return libvirt.DOMAIN_RUNNING, nil
        
      } else {
        err = fmt.Errorf("Could not shutdown the VM \"%s\"! State is now "+
          "\"%s\"!", vm.Descriptor.Name, GetStateString(new_state))
        return libvirt.DOMAIN_RUNNING, err
      }
      
    default:
      err = fmt.Errorf("Cannot start the transition of VM \"%s\" to state "+
        "\"%s\". Target state is not allowed!", vm.Descriptor.Name,
        GetStateString(to))
      return state, err
    }
    
  case libvirt.DOMAIN_BLOCKED:
    // TODO: implement
    fallthrough
  case libvirt.DOMAIN_PAUSED:
    // TODO: implement
    fallthrough
  case libvirt.DOMAIN_SHUTDOWN:
    // TODO: implement
    fallthrough
  case libvirt.DOMAIN_CRASHED:
    // TODO: implement
    fallthrough
  case libvirt.DOMAIN_PMSUSPENDED:
    // TODO: implement
    fallthrough
  case libvirt.DOMAIN_SHUTOFF:
    // TODO: implement
    fallthrough
  default:
    err = fmt.Errorf("Illegal state of VM \"%s\": \"%s\".", vm.Descriptor.Name,
      GetStateString(state))
    return state, err
  }
  
}

// Shutdown tries to shutdown the VM on which the method is executed. Since a
// virtual machine may hang or even ignore the request to shutdown gracefully,
// you can specify (force parameter) wheter you want to force the shutdown
// (plug the power cord) after several "normal" tries. The method returns
// the previous state of the VM.
func (vm *VM) Shutdown(force bool, timeout int) (libvirt.DomainState, error) {
  former_state, _, err := vm.Instance.GetState()
  if err != nil {
    err = fmt.Errorf("Could not retrieve the state of the VM \"%s\"; "+
      "Error was: %v", vm.Descriptor.Name, err)
    return libvirt.DOMAIN_NOSTATE, err
  }
  
  Logger.Debugf("Initial state of VM \"%s\" is \"%s\"", vm.Descriptor.Name, 
    GetStateString(former_state))
  
  switch former_state {

  case libvirt.DOMAIN_RUNNING:
    round_seconds := 0.33 * float64(timeout*60)
    new_state := libvirt.DOMAIN_RUNNING

    // if the virtual machine seems to not react to the first shutdown request,
    // repeatedly send further requests to gracefully shutdown
    for i := 0; i < 3; i++ {
      before := time.Now()
      
      Logger.Debugf("Sending shutdown request to VM \"%s\".",
        vm.Descriptor.Name)
      err = vm.Instance.Shutdown() // returns instantly
      if err != nil {
        // we need to cast to specific libvirt error, since the VM might
        // be in a shutoff state since last check. If this is the case, we
        // do not want to return an error!
        lverr, ok := err.(libvirt.Error)
        if ok && (lverr.Code == libvirt.ERR_OPERATION_INVALID ||
            strings.Contains(lverr.Message, "domain is not running")) {
          Logger.Debugf("VM \"%s\" was shutdown in the meantime.",
            vm.Descriptor.Name)
          return libvirt.DOMAIN_RUNNING, nil 
        
        } else {
          err = fmt.Errorf("Could not initiate the shutdown request for "+
            "VM \"%s\": %v", vm.Descriptor.Name, err)
          return libvirt.DOMAIN_RUNNING, err
        }
      }
      
      Logger.Debugf("Waiting vor the VM \"%s\" to shutdown.", 
        vm.Descriptor.Name)
      for true {
        
        // sleep some seconds
        time.Sleep(5 * time.Second)
        
        new_state, _, err = vm.Instance.GetState()
        if err != nil {
          err = fmt.Errorf("Could not re-retrieve the state of the VM \"%s\". "+
            "Trying again...: %v", vm.Descriptor.Name, err)
          Logger.Warn(err)
        }
        
        if new_state == libvirt.DOMAIN_SHUTOFF {
          return libvirt.DOMAIN_RUNNING, nil
        }
        
        // if we waited longer since 33% of the timeout, try sending the
        // shutdown request again
        after := time.Now()
        duration := after.Sub(before) // int64 nanosecods
        max_round_duration := time.Duration(round_seconds)*time.Second
        if duration > max_round_duration {
          Logger.Debug("Beginning next round.")
          break
        }
      }  
    }
    
    // could not shutdown the VM gracefully, force?
    if force {
      Logger.Debugf("Destroying the VM \"%s\" since it could not be shutdown "+
        "gracefully.", vm.Descriptor.Name)
      err = vm.Instance.Destroy()
      if err != nil {
        err = fmt.Errorf("Could not destroy the VM \"%s\": %v",
          vm.Descriptor.Name, err)
        return libvirt.DOMAIN_RUNNING, err
      }
      return libvirt.DOMAIN_RUNNING, nil
    } else {
      err = fmt.Errorf("Could not shutdown the VM \"%s\"! State is now \"%s\"!", 
        vm.Descriptor.Name, GetStateString(new_state))
      return libvirt.DOMAIN_RUNNING, err
    }
  
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

// Start is a simple wrapper method around the libvirt.Create function that
// tries the startup the VM multiple times before returning an error.
func (vm *VM) Start() error {
  var err error
  for i := 0; i < 3; i++ {
    err = vm.Instance.Create()
    if err == nil {
      return nil
    } else {
      Logger.Errorf("Could not boot up the VM \"%s\": %v", vm.Descriptor.Name,
        err)
      time.Sleep(5 * time.Second)
    }
  }
  return err
}

func (vm *VM) GetCurrentStateString() (string, error) {
  state, _, err := vm.Instance.GetState()
  if err != nil {
    err = fmt.Errorf("Could not retrieve the state of the VM %s; "+
      "Error was: %v", vm.Descriptor.Name, err)
    return "DOMAIN_NOSTATE", err
  }
  
  return GetStateString(state), nil
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
