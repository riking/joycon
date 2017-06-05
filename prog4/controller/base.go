package controller

import (
	"fmt"

	"github.com/riking/joycon/prog4/jcpc"
)

type base struct {
	output jcpc.Output
	ui     jcpc.Interface

	curState  jcpc.CombinedState
	prevState jcpc.CombinedState
}

func Pro(jc jcpc.JoyCon, ui jcpc.Interface) jcpc.Controller {
	panic("NotImplemented")
}

func (c *base) BindToOutput(o jcpc.Output) {
	c.output = o
}

func (c *base) OnFrame() {
}

func (c *base) Close() error {
	return nil
}

func (c *base) dispatchUpdates() {
	c.output.BeginUpdate()
	buttonDiff := c.prevState.Buttons.DiffMask(c.curState.Buttons)
	for _, bu := range jcpc.ButtonList {
		if buttonDiff.Get(bu) {
			c.output.ButtonUpdate(bu, c.curState.Buttons.Get(bu))
		}
	}
	for i := 0; i < 4; i++ {
		if c.prevState.RawSticks[i/2][i%2] != c.curState.RawSticks[i/2][i%2] {
			c.output.StickUpdate(jcpc.AxisID(i), int8(c.curState.RawSticks[i/2][i%2]-0x80))
		}
	}
	if c.curState.Gyro != jcpc.GyroZero {
		c.output.GyroUpdate(c.curState.Gyro[0])
		c.output.GyroUpdate(c.curState.Gyro[1])
		c.output.GyroUpdate(c.curState.Gyro[2])
	}
	err := c.output.FlushUpdate()
	if err != nil {
		fmt.Println("Output error:", err)
	}
}
