package controller

import (
	"sync"
	"time"

	"github.com/riking/joycon/prog4/jcpc"
)

type pro struct {
	base

	mu sync.Mutex
	jc jcpc.JoyCon

	lastUpdate time.Time

	prevBattery int8

	stdTransitionDelay int8
}

func Pro(jc jcpc.JoyCon, ui jcpc.Interface) jcpc.Controller {
	return &pro{
		jc: jc,
		base: base{
			ui: ui,
		},
		stdTransitionDelay: 3,
	}
}

func (c *pro) Rumble(data []jcpc.RumbleData) {
	c.jc.Rumble(data)
}

func (c *pro) JoyConUpdate(jc jcpc.JoyCon, flags int) {
	if flags&jcpc.NotifyInput != 0 {
		c.update()
	}

	if flags&jcpc.NotifyConnection != 0 {
		if jc.IsStopping() {
			c.ui.RemoveController(c)
		}
	}
}

func (c *pro) update() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.prevState = c.curState
	c.curState = jcpc.CombinedState{}
	c.jc.ReadInto(&c.curState, true)

	c.dispatchUpdates()

	if c.stdTransitionDelay > 0 {
		c.stdTransitionDelay--
		if c.stdTransitionDelay == 0 {
			c.jc.ChangeInputMode(jcpc.InputStandard)
		}
	}
}
