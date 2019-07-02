/*
* Copyright (c) 2019 Jonas R. <joroec@gmx.net>
* Licensed under the MIT License. You have obtained a copy of the License at the
* "LICENSE" file in this repository.
*/
package main

import(
  "github.com/joroec/virsnap/cmd"
)

/*
* Main entry point of the application. Just pass execution to command line
* parser.
*/
func main() {
    cmd.RootCmd.Execute()
}
