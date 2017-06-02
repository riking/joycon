package controller

import "github.com/riking/joycon/prog4/jcpc"

type base struct {
	output jcpc.Output

	curState  jcpc.CombinedState
	prevState jcpc.CombinedState
}

func (c *base) BindToOutput(o jcpc.Output) {
	c.output = o
}

func (c *base) OnFrame() {
}

func (c *base) Close() error {
	return nil
}
