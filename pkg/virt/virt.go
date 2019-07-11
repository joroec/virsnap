// Copyright (c) 2019 Jonas R. <joroec@gmx.net>
// Licensed under the MIT License. You have obtained a copy of the License at
// the "LICENSE" file in this repository.

// Package virt implements high-level functions for handling virtual machines
// (VMS) that use the more low-level libvirt functions internally.
package virt

import (
  "os"
  
  "github.com/sirupsen/logrus"
)

// Log is the logrus object that is used of this package to print warning
// messages on the console. The default Logger is configured to use stdout for
// logging and sets a default level of Info. If you want to disable the logging
// output of this package, you need to set a member of the Logger from your
// own code:
// virt.Logger.Out = ioutil.Discard
// After this, no logging output is done anymore.
var Logger = logrus.New()

// init is a special golang function that is called exactly once regardless
// how often the package is imported.
func init() {
  Logger.Out = os.Stdout
  Logger.SetLevel(logrus.WarnLevel)
}
