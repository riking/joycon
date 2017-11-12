package controller

import (
	"sync"
	"time"

	"github.com/riking/joycon/prog4/jcpc"
)

type two struct {
	base

	mu    sync.Mutex
	left  jcpc.JoyCon
	right jcpc.JoyCon

	lastLeft  time.Time
	lastRight time.Time

	stdTransitionDelay int8
}

func TwoJoyCons(left, right jcpc.JoyCon, ui jcpc.Interface) jcpc.Controller {
	return &two{
		left:  left,
		right: right,
		base: base{
			ui: ui,
		},
		stdTransitionDelay: 3,
	}
}

func (c *two) Rumble(data []jcpc.RumbleData) {
	c.left.Rumble(data)
	c.right.Rumble(data)
}

func (c *two) JoyConUpdate(jc jcpc.JoyCon, flags int) {
	isLeft := jc == c.left

	if flags&jcpc.NotifyInput != 0 {
		c.mu.Lock()
		c.prevState = c.curState
		c.curState = c.prevState
		if isLeft {
			c.left.ReadInto(&c.curState, false)
			c.lastLeft = time.Now()
		} else {
			c.right.ReadInto(&c.curState, true)
			c.lastRight = time.Now()
		}

		c.dispatchUpdates()
		c.handleTransition()
		c.mu.Unlock()
	}

	if flags&jcpc.NotifyConnection != 0 {
		if jc.IsStopping() {
			c.ui.RemoveController(c)
		}
	}
}

func (c *two) handleTransition() {
	if c.stdTransitionDelay > 0 {
		c.stdTransitionDelay--
		if c.stdTransitionDelay == 0 {
			go c.left.ChangeInputMode(jcpc.InputStandard)
			go c.right.ChangeInputMode(jcpc.InputStandard)
		}
	}
}
