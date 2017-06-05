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

func NewConsoleFactory() jcpc.OutputFactory {
	return &consoleFactory{}
}

func (f *consoleFactory) New(bool) (jcpc.Output, error) {
	f.i++
	return &consoleOutput{i: f.i}, nil
}

func (c *consoleOutput) OnFrame() {
}

func (c *consoleOutput) BeginUpdate() {
}

func (c *consoleOutput) ButtonUpdate(bu jcpc.ButtonID, state bool) {
	pressed := "pressed"
	if !state {
		pressed = "released"
	}
	fmt.Printf("[Controller %d] %s %s\n", c.i, bu.String(), pressed)
}

func (c *consoleOutput) StickUpdate(axis int, value int8) {

}

func (c *consoleOutput) GyroUpdate(d [3]jcpc.GyroFrame) {

}

func (c *consoleOutput) Flush() error {
	return nil
}

func (c *consoleOutput) Close() error {
	return nil
}
