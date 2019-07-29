// Copyright (c) 2019 The virnsnap authors. See file "AUTHORS".
// Licensed under the MIT License. You have obtained a copy of the License at
// the "LICENSE" file in this repository.

// Package fs implements helper functions for handling filesystem related
// tasks.
package fs

import (
	"fmt"
	"os"

	"golang.org/x/sys/unix"
)

// EnsureDirectory is a helper function that ensures path points to a directory.
// If the directory does not exist, create it. The directory needs to be
// accessible and writeable so the check can pass.
func EnsureDirectory(path string) error {
	stat, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			err = os.Mkdir(path, 0700)
			if err != nil {
				err = fmt.Errorf("could not create directory %s: %s", path, err)
				return err
			}
		} else {
			return err
		}
	}

	// check if stat denotes a directory
	if !stat.IsDir() {
		err = fmt.Errorf("%s does not point to a directory", path)
		return err
	}

	// check permissions for this directory
	err = unix.Access(path, unix.R_OK|unix.W_OK)
	if err != nil {
		err = fmt.Errorf("could not access directory %s: %s", path, err)
		return err
	}

	return nil
}
