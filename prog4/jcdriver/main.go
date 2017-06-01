package main

import "runtime"

func main() {
	// need 1 thread per blocked cgo call
	runtime.GOMAXPROCS(8 + runtime.NumCPU())
}
