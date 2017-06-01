package jcpc

import "image/color"

type JoyCon interface {
	BindToController(Controller)
	Serial() string
	Type() JoyConType

	Buttons() ButtonState
	Battery() int8

	// Valid returns have alpha=255. If alpha=0 the value is not yet available.
	CaseColor() color.RGBA
	ButtonColor() color.RGBA

	Rumble(d []RumbleData)

	Close() error
}

type Controller interface {
	JoyConUpdate(JoyCon)
	BindToOutput(Output)

	// forwards to each JoyCon
	Rumble(d []*RumbleData)

	Close() error
}

type Output interface {
	// The Controller should call several *Update() methods followed by Flush().
	ButtonUpdate(b ButtonState, changed ButtonState)
	StickUpdate(axis int, value int8)
	GyroUpdate(axis int, value int8)
	Flush() error
	Close() error
}

type Interface interface {
	PairingPulse(jc JoyCon)
}