package output

import (
	"fmt"

	"github.com/riking/joycon/prog4/jcpc"
)

type consoleOutput struct {
	i int
}

type consoleFactory struct {
	i int
}


func NewConsole(t jcpc.JoyConType, playerNum int) (jcpc.Output, error) {
	return &consoleOutput{i: playerNum}, nil
}

func (c *consoleOutput) BeginUpdate() error {
	return nil
}

func (c *consoleOutput) ButtonUpdate(bu jcpc.ButtonID, state bool) {
	pressed := "pressed"
	if !state {
		pressed = "released"
	}
	fmt.Printf("[Controller %d] %s %s\n", c.i, bu.String(), pressed)
}

func (c *consoleOutput) StickUpdate(axis jcpc.AxisID, value int8) {

}

func (c *consoleOutput) GyroUpdate(d jcpc.GyroFrame) {}

func (c *consoleOutput) FlushUpdate() error {
	return nil
}

func (c *consoleOutput) OnFrame() {}

func (c *consoleOutput) Close() error {
	return nil
}
