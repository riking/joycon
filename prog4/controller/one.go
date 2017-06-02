package controller

import "github.com/riking/joycon/prog4/jcpc"

type oneJoyConController struct {
	base

	only        jcpc.JoyCon
	out         jcpc.Output
	prevButtons jcpc.ButtonState
}

func OneJoyCon(jc jcpc.JoyCon) jcpc.Controller {
	//return &oneJoyConController{only: jc}
	panic("notImplemented")
}

func Pro(jc jcpc.JoyCon) jcpc.Controller {
	panic("notImplemented")
}
