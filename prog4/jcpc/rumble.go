package jcpc

type RumbleData struct {
	Data [8]byte
	// number of frames that Data remains the same
	Time int
}

var RumbleDataNeutral = RumbleData{[8]byte{0, 1, 0x40, 0x40, 0, 1, 0x40, 0x40}, 8}

type GyroFrame [6]int16

var GyroZero [3]GyroFrame
