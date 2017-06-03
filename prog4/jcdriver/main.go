package main

import (
	"runtime"

	"github.com/riking/joycon/prog4/consoleiface"
	"github.com/riking/joycon/prog4/output"
)

func main() {
	// need 1 thread per blocked cgo call
	runtime.GOMAXPROCS(8 + runtime.NumCPU())

	factory := output.NewConsoleFactory()
	iface := consoleiface.New(factory)
	iface.Run()
}
