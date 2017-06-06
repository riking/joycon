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

/*

00000040  08 00 95 22 47 ab[34 02  86 62 b8 6b]61 3b 29 98
00000050  66 12 22 e2 b0 71 f9 26  3e d4 ee b0

00000000  68 00 95 22 23 8f[34 02  86 62 b8 6b]7d 40 b2 a6
00000010  6d d8 93 60 f7 da ff e2  22 e3 c8 61

00000000  68 00 95 22 40 a2[34 02  86 62 b8 6b]44 ac d8 fa
00000010  03 45 f7 1a fd a3 1e f6  7b c1 1f a2

95 22 = calibration data, probably
 */

type calibrationData struct {

}