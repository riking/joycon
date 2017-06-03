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

	leftPrevBattery  int8
	rightPrevBattery int8
}

func TwoJoyCons(left, right jcpc.JoyCon) jcpc.Controller {
	return &two{
		left:  left,
		right: right,
	}
}

func (c *two) Rumble(data []jcpc.RumbleData) {
	c.left.Rumble(data)
	c.right.Rumble(data)
}

func (c *two) JoyConUpdate(jc jcpc.JoyCon) {
	isLeft := jc == c.left

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
	c.mu.Unlock()

	if bothReady {
		c.updateBoth()
	} else {
		// wait for other controller to update
		return
	}
}

func (c *two) updateBoth() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.prevState = c.curState
	c.curState = jcpc.CombinedState{}
	c.left.ReadInto(&c.curState, false)
	c.right.ReadInto(&c.curState, true)

	buttonDiff := c.prevState.Buttons.DiffMask(c.curState.Buttons)
	for _, bu := range jcpc.ButtonList {
		if buttonDiff.Get(bu) {
			c.output.ButtonUpdate(bu, c.curState.Buttons.Get(bu))
		}
	}
	for i := 0; i < 4; i++ {

	}
	c.output.Flush()
}
