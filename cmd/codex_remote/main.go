// Package main starts the DeskSlice server.
package main

import "flag"

// main is the entrypoint for the DeskSlice server.
func main() {
	debug := flag.Bool("debug", false, "Enable verbose debug logging")
	flag.Parse()

	if err := run(*debug); err != nil {
		logFatal(err)
	}
}
