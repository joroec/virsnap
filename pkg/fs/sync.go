// Copyright (c) 2019 The virnsnap authors. See file "AUTHORS".
// Licensed under the MIT License. You have obtained a copy of the License at
// the "LICENSE" file in this repository.

// Package fs implements helper functions for handling filesystem related
// tasks.
package fs

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/joroec/virsnap/pkg/instrument/log"
)

// Sync is a minimal and opinionated wrapper around a call to
// "rsync -avp <source> <destination>"
func Sync(source string, destination string, logger log.Logger) error {
	// find rsync in path
	rsyncPath, err := exec.LookPath("rsync")
	if err != nil {
		err = fmt.Errorf("could not find rsync: %v", err)
		return err
	}
	logger.Debugf("found rsync at '%s'", rsyncPath)

	// call rsync and show rsync's output
	logger.Debugf("executing command 'rsync -avP %s %s'", source, destination)
	cmd := exec.Command(rsyncPath, "-avP", source, destination)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// start and wait for command to complete, return err if exists with exit
	// code inequal to zero.
	return cmd.Run()
}
