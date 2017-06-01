package joycon

import (
	"sync"

	"github.com/GeertJohan/go.hid"
	"github.com/riking/joycon/prog4/jcpc"
	"image/color"
)

const (
	// the joycon pushes updates to button presses with the 0x3F command.
	modeButtonPush = iota
	// the host requests the current status with a 0x01 command.
	modeInputPolling
	// the joycon pushes the current state at 60Hz.
	modeInputPushing
)

type joyconBluetooth struct {
	hidHandle *hid.Device
	done chan struct{}

	serial string
	side jcpc.JoyConType

	mu sync.Mutex

	controller jcpc.Controller

	battery int8
	raw_stick_v uint8
	raw_stick_h uint8
	buttons jcpc.ButtonState

	haveColors bool
	caseColor color.RGBA
	buttonColor color.RGBA

	rumbleQueue   []jcpc.RumbleData
	rumbleCurrent jcpc.RumbleData
	rumbleTimer   int8

	gyro [12]int16

	subcommandQueue [][]byte
}

func NewFromBluetooth(hidHandle *hid.Device, side jcpc.JoyConType) (jcpc.JoyCon, error) {
	var err error
	jc := &joyconBluetooth{
		hidHandle: hidHandle,
		done: make(chan struct{}),
	}
	jc.serial, err = hidHandle.SerialNumberString()
	if err != nil {
		return nil, err
	}
	jc.side = side
	jc.controller = nil
	jc.haveColors = false
	return jc, nil
}

func (jc *joyconBluetooth) Serial() string {
	return jc.serial
}

func (jc *joyconBluetooth) Type() jcpc.JoyConType {
	return jc.side
}

func (jc *joyconBluetooth) Buttons() jcpc.ButtonState {
	return jc.buttons
}

func (jc *joyconBluetooth) Battery() int8 {
	return jc.battery
}

func (jc *joyconBluetooth) CaseColor() color.RGBA {
	return jc.caseColor
}

func (jc *joyconBluetooth) ButtonColor() color.RGBA {
	return jc.caseColor
}

func (jc *joyconBluetooth) Rumble(d []jcpc.RumbleData) {
	jc.mu.Lock()
	defer jc.mu.Unlock()

	jc.rumbleQueue = append(jc.rumbleQueue, d...)
}

func (jc *joyconBluetooth) Close() error {
	disconnectCommand := []byte{6}
	jc.queueSubcommand(disconnectCommand)
	return nil
}

// mu must be held
func (jc *joyconBluetooth) getNextRumble() (int8, [8]byte, bool) {
	if jc.rumbleCurrent.Time > 0 {
		jc.rumbleCurrent.Time--
		return jc.rumbleTimer, jc.rumbleCurrent.Data, false
	}
	jc.rumbleTimer++
	if jc.rumbleTimer == 16 {
		jc.rumbleTimer = 0
	}
	if len(jc.rumbleQueue) > 0 {
		jc.rumbleCurrent = jc.rumbleQueue[0]
		jc.rumbleQueue = jc.rumbleQueue[1:]
	} else {
		jc.rumbleCurrent = jcpc.RumbleDataNeutral
	}
	return jc.rumbleTimer, jc.rumbleCurrent.Data, true
}

// mu must be held
func (jc *joyconBluetooth) getNextSubcommand() []byte {
	if len(jc.subcommandQueue) > 0 {
		r := jc.subcommandQueue[0]
		jc.subcommandQueue = jc.subcommandQueue[1:]
		return r
	}
	return nil
}

func (jc *joyconBluetooth) sendRumble() {
	jc.mu.Lock()
	timer, data, forceUpdate := jc.getNextRumble()
	subc := jc.getNextSubcommand()
	jc.mu.Unlock()

	if !forceUpdate && subc == nil {
		// nothing to do
		return
	}


}

func (jc *joyconBluetooth) queueSubcommand(data []byte) {
	jc.mu.Lock()
	jc.subcommandQueue = append(jc.subcommandQueue, data)
	jc.mu.Unlock()
}