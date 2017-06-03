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
	RawSticks(axis AxisID) [2]byte
	Battery() int8
	ReadInto(out *CombinedState, includeGyro bool)

	EnableIMU(status bool)

	// Valid returns have alpha=255. If alpha=0 the value is not yet available.
	CaseColor() color.RGBA
	ButtonColor() color.RGBA

	Rumble(d []RumbleData)
	SendCustomSubcommand(d []byte)

	OnFrame()

	Close() error
}

type Controller interface {
	JoyConUpdate(JoyCon)
	BindToOutput(Output)

	// forwards to each JoyCon
	Rumble(d []RumbleData)

	OnFrame()

	Close() error
}

type Output interface {
	// The Controller should call several *Update() methods followed by Flush().
	ButtonUpdate(b ButtonID, value bool)
	StickUpdate(axis int, value int8)
	GyroUpdate(axis int, value int16)
	Flush() error

	OnFrame()
	Close() error
}

type OutputFactory interface {
	New(isSingleJoyCon bool) (Output, error)
}

type Interface interface {
	JoyConUpdate(JoyCon)
}

/*
gyro data notes

SL/SR on table: [0] =   0, [1] = +15, [2] =   0
SL/SR up      : [0] =   0, [1] = -15, [2] =   0

buttons up    : [0] =  +1, [1] =   0, [2] = +15
buttons down  : [0] =  +1, [1] =   0, [2] = -15

shoulder up:  : [0] = +15, [1] =   0, [2] =   0
shoulder down : [0] = -15, [1] =   0, [2] =   0
*/

type CombinedState struct {
	// 3 frames of 6 values
	Gyro [3][6]int16
	// [left, right][horizontal, vertical]
	RawSticks [2][2]uint8
	Buttons   ButtonState
	// battery is per joycon, can't be combined
}
