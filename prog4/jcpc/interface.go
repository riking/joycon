// Package jcpc (Joy-Con PC) contains constants, interface definitions, and
// short utility functions for the joy-con driver.
package jcpc

import (
	"image/color"
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
	// Ask the JoyCon to disconnect and stay disconnected.
	Shutdown()
	// Must be of type github.com/GeertJohan/go.hid#DeviceInfo
	Reconnect(hidDeviceInfo interface{})

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

type OutputFactory func(t JoyConType, playerNum int, remap InputRemappingOptions) (Output, error)

type Interface interface {
	JoyConNotify
	RemoveController(c Controller)
}

// BluetoothManager provides an interface to the OS bluetooth stack.
type BluetoothManager interface {
	// Call StartDiscovery when the UI enters a "change controller
	// order/layout" screen.
	StartDiscovery()
	// Call StopDiscovery when the UI exits a "change controller order/layout"
	// screen. Paired controllers set up for auto-reconnect will still generate
	// device connect events.
	StopDiscovery()

	// Check for devices already connected to the system, send notifications
	// on NotifyChannel, and subscribe to future changes.
	InitialScan()

	// The UI code should call this after a L+R press to ensure clean
	// auto-reconnect.
	SavePairingInfo(mac [6]byte)
	// The UI code must provide a way for the user to reset auto-reconnect
	// records, which (Linux) will occur whenever the Joy-Con is connected to a
	// different Switch.
	DeletePairingInfo()

	NotifyChannel() <-chan BluetoothDeviceNotification
}

type BluetoothDeviceNotification struct {
	MAC       [6]byte
	MACString string
	// false if this is a disconnect event
	Connected bool
	NewDevice bool
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
	InputIRPolling        InputMode = 0
	InputIRPollingUnused            = 1
	InputIRPollingSpecial           = 2
	InputMCUUpdate                  = 0x23 // not fully known
	InputStandard                   = 0x30
	InputNFC                        = 0x31
	InputUnknown33                  = 0x33
	InputUnknown35                  = 0x35
	InputLazyButtons      InputMode = 0x3F
	InputActivePolling              = 0x13F // pseudo-mode, driver only
)

func (i InputMode) NeedsEmptyRumbles() bool {
	return i == InputActivePolling
}

//Options specifies Options for changing the programms behavior (for example obtained via cli-flags)
type Options struct {
	InputRemapping InputRemappingOptions
}

//InputRemappingOptions specifies if and how Buttons or Axes should be remapped
//currently only Axis-Inversion is implemented
type InputRemappingOptions struct {
	InvertedAxes []AxisID
}
