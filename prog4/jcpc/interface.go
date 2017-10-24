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
	// Indexed by [left,right][x,y]
	// Prefer use of ReadInto() for stick data
	RawSticks() [2][2]uint16
	Battery() (int8, bool)
	ReadInto(out *CombinedState, includeGyro bool)

	ChangeInputMode(mode InputMode) bool // returns false if impossible
	EnableGyro(status bool)
	SPIRead(addr uint32, len byte) ([]byte, error)
	SPIWrite(addr uint32, p []byte) error

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
	StickUpdate(axis AxisID, value int16)
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
	// range is -0x7FF to +0x7FF
	AdjSticks [2][2]int16
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

type InputMode int

const (
	InputIRPolling InputMode = 0
	InputIRPollingUnused = 1
	InputIRPollingSpecial = 2
	InputMCUUpdate = 0x23 // not fully known
	InputStandard = 0x30
	InputNFC = 0x31
	InputUnknown33 = 0x33
	InputUnknown35 = 0x35
	InputLazyButtons InputMode = 0x3F

	InputActivePolling = 0x13F // pseudo-mode, driver only
)

func (i InputMode) NeedsEmptyRumbles() bool {
	return i == InputActivePolling
}
