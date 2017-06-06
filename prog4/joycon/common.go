package joycon

import (
	"context"
	"math"

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
	// fake struct to get started
	hzMin, hzMax  int8
	vtMin, vtMax  int8
	hzNeu, hzDead int8
	vtNeu, vtDead int8
}

const magnitudeMax = 1.05

var fakeCalibrationData = calibrationData{
	hzMin: -80, hzMax: 80,
	vtMin: 80, vtMax: 80,
	hzNeu: 0,
	vtNeu: 0,
}

func (_c *calibrationData) Adjust(rawStick [2]uint8) [2]int8 {
	c := _c
	if c == nil {
		c = &fakeCalibrationData
	}
	var hzRaw = 0x80 - int16(rawStick[0])
	var vtRaw = 0x80 - int16(rawStick[1])

	hzOffset := hzRaw - int16(c.hzNeu)
	vtOffset := vtRaw - int16(c.vtNeu)

	var hzStretch, vtStretch float64
	if hzOffset > 0 {
		hzStretch = float64(hzOffset) / float64(c.hzMax-c.hzNeu)
	} else {
		hzStretch = float64(hzOffset) / float64(c.hzNeu-c.hzMin)
	}
	if vtOffset > 0 {
		vtStretch = float64(vtOffset) / float64(c.vtMax-c.vtNeu)
	} else {
		vtStretch = float64(vtOffset) / float64(c.vtNeu-c.vtMin)
	}

	magnitude := hzStretch*hzStretch + vtStretch*vtStretch
	if magnitude > magnitudeMax {
		angle := math.Atan2(vtStretch, hzStretch)
		hzStretch = math.Cos(angle) * magnitudeMax
		vtStretch = math.Sin(angle) * magnitudeMax
		if hzStretch > 1 {
			hzStretch = 1
		}
		if vtStretch > 1 {
			vtStretch = 1
		}
	}

	return [2]int8{int8(hzStretch * 127), int8(vtStretch * 127)}
}
