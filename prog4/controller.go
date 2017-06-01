package prog4

import "github.com/riking/joycon/prog4/jcpc"

type twoJoyConController struct {
	left  jcpc.JoyCon
	right jcpc.JoyCon
	out   jcpc.Output

	combinedButtons jcpc.ButtonState
}

func TwoJoyCons(left, right jcpc.JoyCon) jcpc.Controller {
	return &twoJoyConController{
		left:  left,
		right: right,
		out:   nil,
	}
}

type oneJoyConController struct {
	jcpc.JoyCon
	out         jcpc.Output
	prevButtons jcpc.ButtonState
}

func OneJoyCon(jc jcpc.JoyCon) jcpc.Controller {
	return &oneJoyConController{JoyCon: jc}
}

func (c *oneJoyConController) JoyConUpdate(jc jcpc.JoyCon) {
	if jc != c.JoyCon {
		return
	}
	if c.out == nil {
		return
	}

}
