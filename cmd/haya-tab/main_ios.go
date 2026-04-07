//go:build ios

package main

import "C"

//export WailsIOSMain
func WailsIOSMain() {
	// Do not lock the goroutine to the current OS thread on iOS.
	main()
}
