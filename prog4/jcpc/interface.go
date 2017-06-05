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
	SPIRead(addr uint32, len byte) ([]byte, error)

	// Valid returns have alpha=255. If alpha=0 the value is not yet available.
	CaseColor() color.RGBA
	ButtonColor() color.RGBA

	Rumble(d []RumbleData)
	SendCustomSubcommand(d []byte)

	OnFrame()

	Close() error
}

type Controller interface {
	JoyConNotify
	BindToOutput(Output)

	// forwards to each JoyCon
	Rumble(d []RumbleData)

	OnFrame()

	Close() error
}

// Output represents an OS-level event sink for a Controller object.
// The Controller should call BeginUpdate(), then several *Update() methods, followed by FlushUpdate().
type Output interface {
	BeginUpdate() error
	ButtonUpdate(b ButtonID, value bool)
	StickUpdate(axis AxisID, value int8)
	GyroUpdate(vals GyroFrame)
	FlushUpdate() error

	OnFrame()
	Close() error
}

type OutputFactory func(t JoyConType, playerNum int) (Output, error)

type Interface interface {
	JoyConNotify
	RemoveController(c Controller)
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
	Gyro [3]GyroFrame
	// [left, right][horizontal, vertical]
	RawSticks [2][2]uint8
	Buttons   ButtonState
	// battery is per joycon, can't be combined
}

const (
	NotifyInput = 1 << iota
	NotifyConnection
	NotifyBattery
)

type JoyConNotify interface {
	JoyConUpdate(jc JoyCon, flags int)
}
