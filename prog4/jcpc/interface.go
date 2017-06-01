package jcpc

import (
	"image/color"

	"github.com/GeertJohan/go.hid"
)

type JoyCon interface {
	BindToController(Controller)
	BindToInterface(Interface)
	Serial() string
	Type() JoyConType

	// Returns true if a reconnect is needed - a communication error has occurred, and
	// Close() / Shutdown() have not been called.
	WantsReconnect() bool
	// Returns true if Close() or Shutdown() have been called.
	IsStopping() bool
	// Ask the JoyCon to disconnect. This
	Shutdown()
	Reconnect(info *hid.DeviceInfo)

	Buttons() ButtonState
	Axis(axis AxisID) int16
	Battery() int8

	// Valid returns have alpha=255. If alpha=0 the value is not yet available.
	CaseColor() color.RGBA
	ButtonColor() color.RGBA

	Rumble(d []RumbleData)
	SetPlayerLights(pattern byte)
	SendCustomSubcommand(d []byte)

	OnFrame()

	Close() error
}

type Controller interface {
	JoyConUpdate(JoyCon)
	BindToOutput(Output)

	// Replace a JoyCon in a Controller.
	// If a JoyCon already occupies the specified slot, Close() is called on it.
	// An error is returned if the slot ID does not exist, or this controller does not support replacement (wired pair).
	SetJoyCon(slot int, jc JoyCon) bool

	// forwards to each JoyCon
	Rumble(d []*RumbleData)

	OnFrame()

	Close() error
}

type Output interface {
	// The Controller should call several *Update() methods followed by Flush().
	ButtonUpdate(b ButtonState, changed ButtonState)
	StickUpdate(axis int, value int8)
	GyroUpdate(axis int, value int8)
	Flush() error

	OnFrame()
	Close() error
}

type Interface interface {
	JoyConUpdate(JoyCon)
}
