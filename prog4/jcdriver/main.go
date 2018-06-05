package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/riking/joycon/prog4/consoleiface"
	"github.com/riking/joycon/prog4/jcpc"
)

//this is needed so we can have one flag multiple times, like --invert LV --invert LH
type arrayFlags []string

func (i *arrayFlags) String() string {
	return ""
}

func (i *arrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

var invertedAxes arrayFlags

func main() {
	flag.Var(&invertedAxes, "invert", "Stick-Axes to invert. --invert LV inverts the vertical axis of the left stick. Can be specified multiple times.")
	flag.Parse()

	// need 1 thread per blocked cgo call
	runtime.GOMAXPROCS(8 + runtime.NumCPU())

	of := getOutputFactory()
	bt, err := getBluetoothManager()
	if err != nil {
		fmt.Println("[FATAL] Could not start up bluetooth manager:", err)
		fmt.Println("You may need different compile options depending on your distribution")
		os.Exit(8)
	}

	opts, err := OptionsFromFlags()
	if err != nil {
		fmt.Println("Error when parsing flags:", err.Error())
		os.Exit(1)
	}
	iface := consoleiface.New(of, bt, *opts)
	iface.Run()

	defer func() {
		fmt.Println("exiting...")
		time.Sleep(2 * time.Second)
	}()
}

// OptionsFormFlags parses the cli-flags into an Options-Struct
func OptionsFromFlags() (*jcpc.Options, error) {
	opts := jcpc.Options{}

	stringToAxis := map[string]jcpc.AxisID{
		"LV": jcpc.Axis_L_Vertical,
		"LH": jcpc.Axis_L_Horiz,
		"RV": jcpc.Axis_R_Vertical,
		"RH": jcpc.Axis_R_Horiz,
	}

	for _, v := range invertedAxes {
		if axisid, exists := stringToAxis[v]; exists {
			opts.InputRemapping.InvertedAxes = append(opts.InputRemapping.InvertedAxes, axisid)
		} else {
			return nil, fmt.Errorf("Unknown Axis %s. Please input only values like (L/R)(V/H)", v)
		}
	}

	return &opts, nil
}
