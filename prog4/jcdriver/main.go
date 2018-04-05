package main

import (
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/riking/joycon/prog4/consoleiface"
)

func main() {
	// need 1 thread per blocked cgo call
	runtime.GOMAXPROCS(8 + runtime.NumCPU())

	of := getOutputFactory()
	bt, err := getBluetoothManager()
	if err != nil {
		fmt.Println("[FATAL] Could not start up bluetooth manager:", err)
		fmt.Println("You may need different compile options depending on your distribution")
		os.Exit(8)
	}

	iface := consoleiface.New(of, bt)
	iface.Run()

	defer func() {
		fmt.Println("exiting...")
		time.Sleep(2 * time.Second)
	}()
}
