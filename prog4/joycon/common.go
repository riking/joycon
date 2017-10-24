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

func decodeUint12(b []byte) (uint16, uint16) {
	d1 := uint16(b[0]) | (uint16(b[1] & 0xF) << 8)
	d2 := uint16(b[1] >> 4) | (uint16(b[2]) << 4)
	return d1, d2
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
	xMaxOff, yMaxOff uint16
	xCenter, yCenter uint16
	xMinOff, yMinOff uint16
}

// side must be TypeLeft or TypeRight; TypeBoth controllers should call this twice
func (c *calibrationData) Parse(b []byte, side jcpc.JoyConType) {
	if side == jcpc.TypeLeft {
		c.xMaxOff, c.yMaxOff = decodeUint12(b[0:3])
		c.xCenter, c.yCenter = decodeUint12(b[3:6])
		c.xMinOff, c.yMinOff = decodeUint12(b[6:9])
	} else {
		c.xCenter, c.yCenter = decodeUint12(b[0:3])
		c.xMinOff, c.yMinOff = decodeUint12(b[3:6])
		c.xMaxOff, c.yMaxOff = decodeUint12(b[6:9])
	}
}

const magnitudeMax = 1.0

var fakeCalibrationData = calibrationData{
	xCenter: 0x800, yCenter: 0x800,
	xMaxOff: 0x400, yMaxOff: 0x400,
	xMinOff: 0x400, yMinOff: 0x400,
}

const desiredRange = 0x7FF

func cachedCalibration(serial string) *calibrationData {
	return nil
}

// Changes raw stick values into [-0x7FF, +0x7FF] values.
func (_c *calibrationData) Adjust(rawStick [2]uint16) [2]int16 {
	c := _c
	if c == nil {
		c = &fakeCalibrationData
	} else if c.xCenter == 4095 {
		c = &fakeCalibrationData
	}

	var out [2]int16
	// careful - need to upcast to int before multiplying
	// 1. convert to signed
	// 2. subtract center value
	// 3. widen to int (!)
	// 4. multiply by desiredRange
	// 5. divide by range-from-center
	if rawStick[0] < c.xCenter {
		out[0] = int16(int((int16(rawStick[0]) - int16(c.xCenter))) * desiredRange / int(c.xMinOff))
	} else {
		out[0] = int16(int((int16(rawStick[0]) - int16(c.xCenter))) * desiredRange / int(c.xMaxOff))
	}
	if rawStick[1] < c.yCenter {
		out[1] = int16(int((int16(rawStick[1]) - int16(c.yCenter))) * desiredRange / int(c.yMinOff))
	} else {
		out[1] = int16(int((int16(rawStick[1]) - int16(c.yCenter))) * desiredRange / int(c.yMaxOff))
	}

	// 6. clamp
	if out[0] > desiredRange || out[0] < -desiredRange || out[1] > desiredRange || out[1] < -desiredRange {
		var modX, modY float64 = float64(out[0]), float64(out[1])
		if modX > desiredRange || modX < -desiredRange {
			// overFactor is slightly over 1 or slightly under -1
			overFactor := modX / desiredRange
			overFactor = math.Copysign(overFactor, 1.0)
			modX /= overFactor
			modY /= overFactor
		}
		if modY > desiredRange || modY < -desiredRange {
			// overFactor is slightly over 1 or slightly under -1
			overFactor := modY / desiredRange
			overFactor = math.Copysign(overFactor, 1.0)
			modX /= overFactor
			modY /= overFactor
		}
		// clamp again in case of fraction weirdness
		if modX > desiredRange {
			modX = desiredRange
		}
		if modX < -desiredRange {
			modX = -desiredRange
		}
		if modY > desiredRange {
			modY = desiredRange
		}
		if modY < -desiredRange {
			modY = -desiredRange
		}
		out[0], out[1] = int16(modX), int16(modY)
	}

	return out
}
