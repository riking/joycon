// +build linux

package main

import (
	"fmt"

	"github.com/riking/joycon/prog4/jcpc"
	"github.com/riking/joycon/prog4/output"
)

func getOutputFactory() jcpc.OutputFactory {
	return func(t jcpc.JoyConType, playerNum int, remap jcpc.InputRemappingOptions) (jcpc.Output, error) {
		switch t {
		case jcpc.TypeLeft:
			return output.NewUInput(output.MappingL, fmt.Sprintf("Half Joy-Con %d", playerNum), remap)
		case jcpc.TypeRight:
			return output.NewUInput(output.MappingR, fmt.Sprintf("Half Joy-Con %d", playerNum), remap)
		case jcpc.TypeBoth:
			return output.NewUInput(output.MappingDual, fmt.Sprintf("Full Joy-Con %d", playerNum), remap)
		}
		panic("bad joycon type")
	}
}
