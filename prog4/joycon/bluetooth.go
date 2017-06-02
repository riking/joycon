package joycon

import (
	"encoding/binary"
	"fmt"
	"image/color"
	"sync"

	"time"

	"github.com/GeertJohan/go.hid"
	"github.com/riking/joycon/prog4/jcpc"
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

	serial string
	side   jcpc.JoyConType

	mu sync.Mutex

	// on a communication error, isAlive is set to false, and Reconnect() can set it back to true.
	isAlive bool
	// isShutdown is set to true on Close() and Shutdown().
	// Shutdown also requests that the JoyCon disconnect from the host.
	isShutdown bool
	mode       int
	controller jcpc.Controller
	ui         jcpc.Interface

	battery   int8
	raw_stick [2][2]byte
	buttons   jcpc.ButtonState
	haveGyro  bool
	gyro      [12]int16

	haveColors  bool
	caseColor   color.RGBA
	buttonColor color.RGBA

	rumbleQueue   []jcpc.RumbleData
	rumbleCurrent jcpc.RumbleData
	rumbleTimer   byte

	subcommandQueue [][]byte
}

func NewBluetooth(hidHandle *hid.Device, side jcpc.JoyConType, ui jcpc.Interface) (jcpc.JoyCon, error) {
	var err error
	jc := &joyconBluetooth{
		hidHandle: hidHandle,
		ui: ui,
	}
	jc.serial, err = hidHandle.SerialNumberString()
	if err != nil {
		return nil, err
	}
	jc.side = side
	jc.controller = nil
	jc.haveColors = false
	jc.mode = modeButtonPush
	jc.isAlive = true
	
	go jc.reader()
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

func (jc *joyconBluetooth) RawSticks(axis jcpc.AxisID) [2]byte {
	if axis == jcpc.Axis_L_Horiz || axis == jcpc.Axis_L_Vertical {
		return jc.raw_stick[0]
	} else {
		return jc.raw_stick[1]
	}
}

func (jc *joyconBluetooth) ReadInto(out *jcpc.CombinedState, includeGyro bool) {
	jc.mu.Lock()
	defer jc.mu.Unlock()

	out.Buttons = out.Buttons.Union(jc.buttons)
	// TODO send CALIBRATED stick data
	if jc.side.IsLeft() {
		out.RawSticks[0] = jc.raw_stick[0]
	}
	if jc.side.IsRight() {
		out.RawSticks[1] = jc.raw_stick[1]
	}

	if includeGyro && jc.haveGyro {
		out.Gyro = jc.gyro
	}
}

func (jc *joyconBluetooth) GyrosInto(buf []int16) bool {
	jc.mu.Lock()
	defer jc.mu.Unlock()

	if !jc.haveGyro {
		return false
	}
	copy(buf, jc.gyro[:])
	return true
}

func (jc *joyconBluetooth) SendCustomSubcommand(d []byte) {
	jc.queueSubcommand(d)
}

func (jc *joyconBluetooth) Rumble(d []jcpc.RumbleData) {
	jc.mu.Lock()
	defer jc.mu.Unlock()

	jc.rumbleQueue = append(jc.rumbleQueue, d...)
}

func (jc *joyconBluetooth) BindToController(c jcpc.Controller) {
	jc.mu.Lock()
	jc.controller = c
	jc.mu.Unlock()
}

func (jc *joyconBluetooth) BindToInterface(c jcpc.Interface) {
	jc.mu.Lock()
	jc.ui = c
	jc.mu.Unlock()
}

func (jc *joyconBluetooth) WantsReconnect() bool {
	jc.mu.Lock()
	defer jc.mu.Unlock()

	return !jc.isAlive && !jc.isShutdown
}

func (jc *joyconBluetooth) IsStopping() bool {
	jc.mu.Lock()
	defer jc.mu.Unlock()

	return jc.isShutdown
}

func (jc *joyconBluetooth) Shutdown() {
	var packet [0x32]byte
	packet[0] = 1
	packet[10] = 6

	jc.mu.Lock()
	defer jc.mu.Unlock()

	if jc.hidHandle != nil {
		jc.hidHandle.Write(packet[:])
	}
	jc.hidHandle.Close()
	jc.hidHandle = nil
	jc.isShutdown = true
	jc.isAlive = false
}

func (jc *joyconBluetooth) Reconnect(dev *hid.DeviceInfo) {
	jc.mu.Lock()
	if jc.isShutdown {
		return
	}
	if jc.hidHandle != nil {
		jc.hidHandle.Close()
	}

	handle, err := dev.Device()
	if err != nil {
		return
	}

	jc.hidHandle = handle
	jc.isAlive = true
	jc.mu.Unlock()
}

func (jc *joyconBluetooth) Close() error {
	jc.mu.Lock()
	defer jc.mu.Unlock()

	jc.hidHandle.Close()
	jc.isAlive = false
	jc.isShutdown = true
	jc.hidHandle = nil
	return nil
}

func (jc *joyconBluetooth) queueSubcommand(data []byte) {
	jc.mu.Lock()
	jc.subcommandQueue = append(jc.subcommandQueue, data)
	jc.mu.Unlock()
}

// OnFrame triggers writes - this way they're rate-limited
func (jc *joyconBluetooth) OnFrame() {
	switch jc.mode {
	case modeButtonPush:
		jc.sendRumble(false)
	case modeInputPolling:
		jc.sendRumble(true)
	case modeInputPushing:
		jc.sendRumble(false)
	}
}

// mu must be held
func (jc *joyconBluetooth) getNextRumble() (byte, [8]byte, bool) {
	if jc.rumbleCurrent.Time > 0 {
		jc.rumbleCurrent.Time--
		return jc.rumbleTimer, jc.rumbleCurrent.Data, false
	}
	needUpdate := true
	jc.rumbleTimer++
	if jc.rumbleTimer == 16 {
		jc.rumbleTimer = 0
	}
	if len(jc.rumbleQueue) > 0 {
		jc.rumbleCurrent = jc.rumbleQueue[0]
		jc.rumbleQueue = jc.rumbleQueue[1:]
	} else {
		if jc.rumbleCurrent.Data == jcpc.RumbleDataNeutral.Data {
			needUpdate = false
			jc.rumbleCurrent.Time = jcpc.RumbleDataNeutral.Time
		} else {
			jc.rumbleCurrent = jcpc.RumbleDataNeutral
		}
	}
	return jc.rumbleTimer, jc.rumbleCurrent.Data, needUpdate
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

func (jc *joyconBluetooth) sendRumble(forceUpdate bool) {
	jc.mu.Lock()
	timer, data, needRumble := jc.getNextRumble()
	subc := jc.getNextSubcommand()
	jc.mu.Unlock()

	if !forceUpdate && !needRumble && subc == nil {
		// nothing to do
		return
	}

	var packet [0x40]byte
	packet[0] = 1
	packet[1] = timer
	copy(packet[2:10], data[:])
	copy(packet[10:], subc)
	// TODO - writePacket function?
	_, err := jc.hidHandle.Write(packet[:])
	if err != nil {
		jc.onReadError(err)
	}
}

func (jc *joyconBluetooth) onReadError(err error) {
	fmt.Printf("[ ERR] JoyCon %p read error: %v\n", jc, err)
	jc.mu.Lock()
	jc.isAlive = false
	jc.hidHandle.Close()
	jc.hidHandle = nil
	jc.mu.Unlock()

	if jc.ui != nil {
		jc.ui.JoyConUpdate(jc)
	}
}

func (jc *joyconBluetooth) fillInput(packet []byte) {
	jc.mu.Lock()

	packet = packet[1:]
	jc.battery = int8((packet[1] & 0xF0) >> 4)
	newButtons := jcpc.ButtonsFromSlice(packet[2:5])
	jc.buttons = newButtons
	if jc.side.IsLeft() {
		jc.raw_stick[0][0] = ((packet[6] & 0x0F) << 4) | ((packet[5] & 0xF0) >> 4)
		jc.raw_stick[0][1] = packet[7]
	}
	if jc.side.IsRight() {
		jc.raw_stick[1][0] = ((packet[9] & 0x0F) << 4) | ((packet[8] & 0xF0) >> 4)
		jc.raw_stick[1][1] = packet[10]
	}
	// packet[11]

	cont := jc.controller
	ui := jc.ui
	jc.mu.Unlock()

	if cont != nil {
		cont.JoyConUpdate(jc)
	}
	if ui != nil {
		ui.JoyConUpdate(jc)
	}
}

func (jc *joyconBluetooth) handleSubcommandReply(_packet []byte) {
	// packetID := _packet[0]
	packet := _packet[1:]

	replyPacketID := packet[12] - 0x80
	if replyPacketID == 0 {
		return
	}

	switch replyPacketID {
	case 0x10: // SPI Flash Read
		jc.handleSPIRead(packet[12:])
	case 0x03:
		fallthrough
	case 0x50:
		fallthrough
	default:
		fmt.Println("got subcommand reply packet:", packet[12:])
	}
}

func (jc *joyconBluetooth) handleButtonPush(packet []byte) {
	if jc.mode != modeButtonPush {
		return
	}

	// translating the buttons is too much of a pain
	// and requires different handling from pro controller
	jc.queueSubcommand([]byte{0})
}

func (jc *joyconBluetooth) reader() {
	var buffer [0x100]byte

	for {
		jc.mu.Lock()
		hidHandle := jc.hidHandle
		isShutdown := jc.isShutdown
		jc.mu.Unlock()

		if isShutdown {
			return
		}
		if hidHandle == nil {
			time.Sleep(15 * time.Millisecond)
			continue
		}

		n, err := hidHandle.Read(buffer[:])
		if err != nil {
			jc.onReadError(err)
			return
		}

		packet := buffer[:n]
		fmt.Println("read packet", packet)
		switch packet[0] {
		case 0x21:
			jc.fillInput(packet)
			jc.handleSubcommandReply(packet)
		case 0x30:
			jc.fillInput(packet)
		case 0x3F:
			jc.handleButtonPush(packet)
		default:
			fmt.Println("[!!] Unknown INPUT packet type ", packet[0])
			fmt.Printf("Packet %02X: %v\n", packet[0], packet)
		}
	}
}

func (jc *joyconBluetooth) handleSPIRead(packet []byte) {
	addr := binary.LittleEndian.Uint32(packet[1:])
	length := packet[5]
	data := packet[6:]

	if addr == 0x6050 && length == 6 {
		jc.mu.Lock()
		jc.haveColors = true
		jc.caseColor.R = data[0]
		jc.caseColor.G = data[1]
		jc.caseColor.B = data[2]
		jc.caseColor.A = 255
		jc.buttonColor.R = data[3]
		jc.buttonColor.G = data[4]
		jc.buttonColor.B = data[5]
		jc.buttonColor.A = 255
		jc.mu.Unlock()
	} else {
		fmt.Printf("SPI flash read @%X len=%d\n%v\n", addr, length, data)
	}

	// TODO callbacks
}
