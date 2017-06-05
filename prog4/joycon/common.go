package joycon

import (
	"context"

	"github.com/riking/joycon/prog4/jcpc"
)

func notify(jc jcpc.JoyCon, flags int, notify ...jcpc.JoyConNotify) {
	for _, v := range notify {
		if v == nil {
			continue
		}
		v.JoyConUpdate(jc, flags)
	}
}

type spiReadCallback struct {
	F       func([]byte, error)
	Ctx     context.Context
	address uint32
	size    byte
}
