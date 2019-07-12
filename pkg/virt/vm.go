// Copyright (c) 2019 The virnsnap authors. See file "AUTHORS".
// Licensed under the MIT License. You have obtained a copy of the License at
// the "LICENSE" file in this repository.

// Package virt implements high-level functions for handling virtual machines
// (VMS) that use the more low-level libvirt functions internally.
package virt

import (
  "fmt"
  "regexp"
  "time"
  "sort"
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

// Transition implements state transitions of the given VM. This method can
// be seen as implementation of an finite state machine (FSM). "to" specifies
// the target state of the VM. "forceShutdown" determines whether the VM should
// be forced to shutoff (plug the cable) after several tries of graceful
// shutdown before returning an error. "timeout" specifies the timeout in
// minutes a VM is allowed to take before forcing shutdown.
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
    
  // HANDLE RUNNING VM ---------------------------------------------------------
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
      
      roundSeconds := 0.33 * float64(timeout*60)
      newState := libvirt.DOMAIN_RUNNING

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
          
          }
          
          err = fmt.Errorf("Could not initiate the shutdown request for "+
            "VM \"%s\": %v", vm.Descriptor.Name, err)
          return libvirt.DOMAIN_RUNNING, err
          
        }
        
        Logger.Debugf("Waiting vor the VM \"%s\" to shutdown.", 
          vm.Descriptor.Name)
        for true {
          time.Sleep(5 * time.Second)
          
          newState, _, err = vm.Instance.GetState()
          if err != nil {
            err = fmt.Errorf("Could not re-retrieve the state of the VM "+
              "\"%s\". Trying again...: %v", vm.Descriptor.Name, err)
            Logger.Warn(err)
          }
          
          if newState == libvirt.DOMAIN_SHUTOFF {
            return libvirt.DOMAIN_RUNNING, nil
          }
          
          // if we waited longer since 33% of the timeout, try sending the
          // shutdown request again
          after := time.Now()
          duration := after.Sub(before) // int64 nanosecods
          maxRoundDuration := time.Duration(roundSeconds)*time.Second
          if duration > maxRoundDuration {
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
        
      }
      
      err = fmt.Errorf("Could not shutdown the VM \"%s\"! State is now "+
        "\"%s\"!", vm.Descriptor.Name, GetStateString(newState))
      return libvirt.DOMAIN_RUNNING, err
      
    default:
      err = fmt.Errorf("Cannot start the transition of VM \"%s\" to state "+
        "\"%s\". Target state is not allowed!", vm.Descriptor.Name,
        GetStateString(to))
      return state, err
    }
  
  // HANDLE CRASHED VM ---------------------------------------------------------
  case libvirt.DOMAIN_CRASHED:
    // treat a crashed like a VM that is shutoff
    fallthrough
  
  // HANDLE SHUTOFF VM ---------------------------------------------------------
  case libvirt.DOMAIN_SHUTOFF:
    // we only need three cases here: Either the VM is already shutdown or
    // the VM should be started. In any other case, the VM needs to be
    // booted up, before the follow-up transition can occur.
    if to == libvirt.DOMAIN_SHUTOFF {
      Logger.Debugf("Domain \"%s\" is already shutoff.", vm.Descriptor.Name)
      return state, nil
    } else if to == libvirt.DOMAIN_RUNNING {
      
      err := vm.Instance.Create()
      if err != nil {
        Logger.Errorf("Could not boot up the VM \"%s\": %v", vm.Descriptor.Name,
          err)
        return state, err
      }
      return state, nil
      
    } else {
      // First Transition: Wait for the VM to be running
      prev, err := vm.Transition(libvirt.DOMAIN_RUNNING, forceShutdown, timeout)
      if err != nil {
        return state, err
      }
      
      if prev != state {
        Logger.Warnf("State of VM \"%s\" has changed in the meantime. Was "+
          "\"%s\", now it is \"%s\"!", vm.Descriptor.Name,
          GetStateString(state), GetStateString(prev))
      }
      
      // Second Transition: Transition to the acutal target state
      prev, err = vm.Transition(to, forceShutdown, timeout)
      if err != nil {
        return state, err
      }
      
      if prev != libvirt.DOMAIN_RUNNING {
        Logger.Warnf("State of VM \"%s\" has changed in the meantime. Was "+
          "\"%s\", now it is \"%s\"!", vm.Descriptor.Name,
          GetStateString(state), GetStateString(prev))
      }
      
      return state, nil
    }
  
  // HANDLE SHUTDOWNING VM -----------------------------------------------------
  case libvirt.DOMAIN_SHUTDOWN:
    // we only need two cases here: Wait for the VM to shutdown and anything
    // else because in any other case than waiting for the VM to shutdown we
    // would need wait nevertheless and then execute the follow-up transition.
    if to == libvirt.DOMAIN_SHUTOFF {
      
      Logger.Debugf("Waiting vor the VM \"%s\" to shutdown.", 
        vm.Descriptor.Name)
      before := time.Now()
      for true {
        time.Sleep(5 * time.Second)
        
        newState, _, err := vm.Instance.GetState()
        if err != nil {
          err = fmt.Errorf("Could not re-retrieve the state of the VM "+
            "\"%s\". Trying again...: %v", vm.Descriptor.Name, err)
          Logger.Warn(err)
        }
        
        if newState == libvirt.DOMAIN_SHUTOFF {
          // returning shutoff, since this will be the future state of the VM.
          // A caller should assume this as previous state, since the VM would
          // have entered the shutoff state without any further intervention.
          return libvirt.DOMAIN_SHUTOFF, nil
        }
        
        after := time.Now()
        duration := after.Sub(before) // int64 nanosecods
        if duration > time.Duration(timeout)*time.Minute {
          Logger.Debugf("Beginning next graceful shutdown round for VM "+
            "\"%s\".", vm.Descriptor.Name)
          break
        }
      }
      
      // returning shutoff, since this will be the future state of the VM.
      // A caller should assume this as previous state, since the VM would
      // have entered the shutoff state without any further intervention.
      return libvirt.DOMAIN_SHUTOFF, nil
      
    }
    
    // In any other case: First Transition: Wait for the VM to be shutoff
    prev, err := vm.Transition(libvirt.DOMAIN_SHUTOFF, forceShutdown, timeout)
    if err != nil {
      // return shutoff, since the VM reaches this state without any further
      // intervention.
      return libvirt.DOMAIN_SHUTOFF, err
    }
    
    if prev != state && prev != libvirt.DOMAIN_SHUTOFF {
      Logger.Warnf("State of VM \"%s\" has changed in the meantime. Was "+
        "\"%s\", now it is \"%s\"!", vm.Descriptor.Name,
        GetStateString(state), GetStateString(prev))
    }
    
    // Second Transition: Transition to the acutal target state
    prev, err = vm.Transition(to, forceShutdown, timeout)
    if err != nil {
      // return shutoff, since the VM reaches this state without any further
      // intervention.
      return libvirt.DOMAIN_SHUTOFF, err
    }
    
    if prev != libvirt.DOMAIN_SHUTOFF {
      Logger.Warnf("State of VM \"%s\" has changed in the meantime. Was "+
        "\"%s\", now it is \"%s\"!", vm.Descriptor.Name,
        GetStateString(state), GetStateString(prev))
    }
    
    // return shutoff, since the VM reaches this state without any further
    // intervention.
    return libvirt.DOMAIN_SHUTOFF, nil
    
  
  // HANDLE PAUSED VM ----------------------------------------------------------
  case libvirt.DOMAIN_PAUSED:
    // we only need three cases here: Either the VM is already paused or
    // the VM should be woken up. In any other case, the VM needs to be
    // woken up, before the follow-up transition can occur.
    if to == libvirt.DOMAIN_PAUSED {
      Logger.Debugf("Domain \"%s\" is already paused.", vm.Descriptor.Name)
      return state, nil
    } else if to == libvirt.DOMAIN_RUNNING {
      
      Logger.Debugf("Resuming domain \"%s\".", vm.Descriptor.Name)
      err = vm.Instance.Resume()
      if err != nil {
        err = fmt.Errorf("Could not resume the VM \"%s\"; Error was: %v",
          vm.Descriptor.Name, err)
        return state, err
      }
      return state, nil
      
    } else {
      // First Transition: Wait for the VM to be resumed
      prev, err := vm.Transition(libvirt.DOMAIN_RUNNING, forceShutdown, timeout)
      if err != nil {
        return state, err
      }
      
      if prev != state {
        Logger.Warnf("State of VM \"%s\" has changed in the meantime. Was "+
          "\"%s\", now it is \"%s\"!", vm.Descriptor.Name,
          GetStateString(state), GetStateString(prev))
      }
      
      // Second Transition: Transition to the acutal target state
      prev, err = vm.Transition(to, forceShutdown, timeout)
      if err != nil {
        return state, err
      }
      
      if prev != libvirt.DOMAIN_RUNNING {
        Logger.Warnf("State of VM \"%s\" has changed in the meantime. Was "+
          "\"%s\", now it is \"%s\"!", vm.Descriptor.Name,
          GetStateString(state), GetStateString(prev))
      }
      
      return state, nil
    }

  // HANDLE PMSUSPENDED VM -----------------------------------------------------
  case libvirt.DOMAIN_PMSUSPENDED:
    // we only need three cases here: Either the VM is already pmsuspended or
    // the VM should be resumed. In any other case, the VM needs to be
    // resumed, before the follow-up transition can occur.
    if to == libvirt.DOMAIN_PMSUSPENDED {
      Logger.Debugf("Domain \"%s\" is already pmsuspended.", vm.Descriptor.Name)
      return state, nil
    } else if to == libvirt.DOMAIN_RUNNING {
      
      Logger.Debugf("Wake up domain \"%s\".", vm.Descriptor.Name)
      err = vm.Instance.PMWakeup(0)
      if err != nil {
        err = fmt.Errorf("Could not wake up the VM \"%s\"; Error was: %v",
          vm.Descriptor.Name, err)
        return state, err
      }
      return state, nil
      
    } else {
      // First Transition: Wait for the VM to be woken up
      prev, err := vm.Transition(libvirt.DOMAIN_RUNNING, forceShutdown, timeout)
      if err != nil {
        return state, err
      }
      
      if prev != state {
        Logger.Warnf("State of VM \"%s\" has changed in the meantime. Was "+
          "\"%s\", now it is \"%s\"!", vm.Descriptor.Name,
          GetStateString(state), GetStateString(prev))
      }
      
      // Second Transition: Transition to the acutal target state
      prev, err = vm.Transition(to, forceShutdown, timeout)
      if err != nil {
        return state, err
      }
      
      if prev != libvirt.DOMAIN_RUNNING {
        Logger.Warnf("State of VM \"%s\" has changed in the meantime. Was "+
          "\"%s\", now it is \"%s\"!", vm.Descriptor.Name,
          GetStateString(state), GetStateString(prev))
      }
      
      return state, nil
    }
  
  // HANDLE BLOCKED VM ---------------------------------------------------------
  case libvirt.DOMAIN_BLOCKED:
    // TODO: What exactly is a blocked VM? Cannot find any documentation. Tell
    // me, if you find some...
    // Wait for the VM to not be blocked anymore and then execute the given
    // transition.
    Logger.Debugf("Waiting vor the VM \"%s\" to not be blocked anymore.", 
      vm.Descriptor.Name)
    before := time.Now()
    for true {
      time.Sleep(5 * time.Second)
      
      newState, _, err := vm.Instance.GetState()
      if err != nil {
        err = fmt.Errorf("Could not re-retrieve the state of the VM "+
          "\"%s\". Trying again...: %v", vm.Descriptor.Name, err)
        Logger.Warn(err)
      }
      
      if newState != libvirt.DOMAIN_BLOCKED {
        // Execute Transition to the acutal target state
        prev, err := vm.Transition(to, forceShutdown, timeout)
        if err != nil {
          return state, err
        }
        
        if prev != newState {
          Logger.Warnf("State of VM \"%s\" has changed in the meantime. Was "+
            "\"%s\", now it is \"%s\"!", vm.Descriptor.Name,
            GetStateString(state), GetStateString(prev))
        }
        
        // return running, since this is the future state if the VM is not
        // blocked any longer
        return libvirt.DOMAIN_RUNNING, nil
        
      }
      
      after := time.Now()
      duration := after.Sub(before) // int64 nanosecods
      if duration > time.Duration(timeout)*time.Minute {
        Logger.Debugf("Beginning next graceful shutdown round for VM "+
          "\"%s\".", vm.Descriptor.Name)
        break
      }
    }
    
    // return running, since this is the future state if the VM is not
    // blocked any longer
    err = fmt.Errorf("Timed out while waiting for VM \"%s\" to not be blocked "+
      " any longer.", vm.Descriptor.Name)
    return libvirt.DOMAIN_RUNNING, err
    
  // HANDLE ANY OTHER CASE -----------------------------------------------------
  default:
    err = fmt.Errorf("Illegal state of VM \"%s\": \"%s\"", vm.Descriptor.Name,
      GetStateString(state))
    return state, err
  }
  
}

// -----------------------------------------------------------------------------

// ListMatchingVMs is a method that allows to retrieve information about
// virtual machines that can be accessed via libvirt. The first parameter
// specifies a slice of regular expressions. Only virtual machines whose name
// matches at least one of the regular expressions are returned. The caller is
// responsible for calling FreeVMs on the returned slice to free any
// buffer in libvirt. The returned VMs are sorted lexically by name.
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
    return nil, fmt.Errorf("No regular expression was specified")
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
  matchedVMs := make([]VM, 0, len(instances))
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
      matchedVM := VM{
        Instance: instance,
        Descriptor: descriptor,
      }
      matchedVMs = append(matchedVMs, matchedVM)
    } else {
      // we do not need the instance here anymore
      err = instance.Free()
      if err != nil {
        err = fmt.Errorf("Could not free the vm %s: %v", descriptor.Name, err)
        Logger.Warn(err)
      }
    }
  }
  
  // sort the VMs according to the name increasingly
  sorter := VMSorter{
    VMs: &matchedVMs,
  }
  sort.Sort(&sorter)
  
  return matchedVMs, nil
}

// -----------------------------------------------------------------------------

// VMSorter is a sorter for sorting snapshots by name lexically.
type VMSorter struct {
  VMs *[]VM
}

func (s *VMSorter) Len() int {
  return len(*s.VMs)
}

func (s *VMSorter) Less(i int, j int) bool {
  return (*s.VMs)[i].Descriptor.Name < (*s.VMs)[j].Descriptor.Name
}

func (s *VMSorter) Swap(i int, j int) {
  (*s.VMs)[i], (*s.VMs)[j] = (*s.VMs)[j], (*s.VMs)[i]
}

// -----------------------------------------------------------------------------

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

// GetCurrentStateString is a helper method that retrieves the current
// state of the VM and returns this state as human readable representation.
func (vm *VM) GetCurrentStateString() (string, error) {
  state, _, err := vm.Instance.GetState()
  if err != nil {
    err = fmt.Errorf("Could not retrieve the state of the VM %s; "+
      "Error was: %v", vm.Descriptor.Name, err)
    return GetStateString(libvirt.DOMAIN_NOSTATE), err
  }
  return GetStateString(state), nil
}

// GetStateString is a helper function that takes a VM state and returns this
// state as human readable representation.
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
