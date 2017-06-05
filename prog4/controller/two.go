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

	lastUpdate time.Time
	leftReady  bool
	rightReady bool

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
		if isLeft {
			c.leftReady = true
		} else {
			c.rightReady = true
		}
		bothReady := c.leftReady && c.rightReady
		if !bothReady {
			if time.Since(c.lastUpdate) > 10*time.Millisecond {
				bothReady = true
			}
		}
		if bothReady {
			c.leftReady = false
			c.rightReady = false
		}
		c.mu.Unlock()

		if bothReady {
			c.updateBoth()
		} else {
			// wait for other controller to update
		}
	}

	if flags&jcpc.NotifyConnection != 0 {
		if jc.IsStopping() {
			c.ui.RemoveController(c)
		}
	}
}

func (c *two) updateBoth() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.prevState = c.curState
	c.curState = jcpc.CombinedState{}
	c.left.ReadInto(&c.curState, false)
	c.right.ReadInto(&c.curState, true)

	c.dispatchUpdates()

	if c.stdTransitionDelay > 0 {
		c.stdTransitionDelay--
		if c.stdTransitionDelay == 0 {
			c.left.ChangeInputMode(jcpc.ModeStandard)
			c.right.ChangeInputMode(jcpc.ModeStandard)
		}
	}
}
