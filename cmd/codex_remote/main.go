// Package main starts the DeskSlice server.
package main

// main is the entrypoint for the DeskSlice server.
func main() {
	if err := run(); err != nil {
		logFatal(err)
	}
}
