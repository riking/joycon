package main

import (
	"runtime"
	"time"
	"fmt"

	"github.com/riking/joycon/prog4/consoleiface"
)

func main() {
	// need 1 thread per blocked cgo call
	runtime.GOMAXPROCS(8 + runtime.NumCPU())

	iface := consoleiface.New(getOutputFactory())
	iface.Run()

	defer func() {
		fmt.Println("exiting...")
		time.Sleep(2*time.Second)
	}()
}
