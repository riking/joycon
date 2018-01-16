package joycon

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"image/color"
	"sync"
	"time"

	"github.com/GeertJohan/go.hid"
	"github.com/pkg/errors"
	"github.com/riking/joycon/prog4/jcpc"
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
	mode       jcpc.InputMode
	controller jcpc.Controller
	ui         jcpc.Interface

	packet1   uint8
	buttons   jcpc.ButtonState
	raw_stick [2][2]uint16       // left, right; x, y
	calib     [2]calibrationData // left, right
	haveGyro  bool
	gyro      [3]jcpc.GyroFrame
	// TODO gyro calibration

	haveColors  bool
	caseColor   color.RGBA
	buttonColor color.RGBA

	rumbleQueue   []jcpc.RumbleData
	rumbleCurrent jcpc.RumbleData
	rumbleTimer   byte

	subcommandQueue [][]byte

	spiReads []spiReadCallback
}

func NewBluetooth(hidHandle *hid.Device, side jcpc.JoyConType, ui jcpc.Interface) (jcpc.JoyCon, error) {
	var err error
	jc := &joyconBluetooth{
		hidHandle: hidHandle,
		ui:        ui,
	}
	jc.serial, err = hidHandle.SerialNumberString()
	if err != nil {
		return nil, err
	}
	jc.side = side
	jc.controller = nil
	jc.haveColors = false
	jc.mode = jcpc.InputLazyButtons
	jc.isAlive = true

	go jc.reader()

	// Read stick calibration and case colors
	// TODO cache this data
	go func() {
		time.Sleep(100 * time.Millisecond)
		_, err := jc.SPIRead(factoryStickCalibStart, factoryStickCalibLen)
		if err != nil {
			fmt.Println("Error:", err)
		}
		_, err = jc.SPIRead(userStickCalibStart, userStickCalibLen)
		if err != nil {
			fmt.Println("Error:", err)
		}
	}()
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

// Battery level and charging state.
//
// 4=full, 3, 2, 1=critical, 0=empty. true=charging.
func (jc *joyconBluetooth) Battery() (int8, bool) {
	return int8(jc.packet1 >> 5), jc.packet1&0x10 != 0
}

func (jc *joyconBluetooth) CaseColor() color.RGBA {
	return jc.caseColor
}

func (jc *joyconBluetooth) ButtonColor() color.RGBA {
	return jc.buttonColor
}

func (jc *joyconBluetooth) EnableGyro(status bool) {
	subcommand := []byte{0x40, 0}
	if status {
		subcommand[1] = 1
	}

	jc.mu.Lock()
	jc.haveGyro = status
	jc.subcommandQueue = append(jc.subcommandQueue, subcommand)
	jc.mu.Unlock()
}

func (jc *joyconBluetooth) RawSticks() [2][2]uint16 {
	return jc.raw_stick
}

func (jc *joyconBluetooth) ChangeInputMode(newMode jcpc.InputMode) bool {
	jc.mu.Lock()
	defer jc.mu.Unlock()

	if newMode == jc.mode {
		return false
	}

	cmd := []byte{0x03, byte(newMode)}
	jc.subcommandQueue = append(jc.subcommandQueue, cmd)
	jc.mode = newMode

	return true
}

func (jc *joyconBluetooth) ReadInto(out *jcpc.CombinedState, includeGyro bool) {
	jc.mu.Lock()
	defer jc.mu.Unlock()

	out.Buttons = out.Buttons.Remove(jc.side).Union(jc.buttons)
	if jc.side.IsLeft() {
		out.AdjSticks[0] = jc.calib[0].Adjust(jc.raw_stick[0])
	}
	if jc.side.IsRight() {
		out.AdjSticks[1] = jc.calib[1].Adjust(jc.raw_stick[1])
	}

	if includeGyro && jc.haveGyro {
		out.Gyro = jc.gyro // TODO 6axis calibration
	}
}

func (jc *joyconBluetooth) SendCustomSubcommand(d []byte) {
	jc.queueSubcommand(d)
}

func (jc *joyconBluetooth) Rumble(d []jcpc.RumbleData) {
	jc.mu.Lock()
	defer jc.mu.Unlock()

	jc.rumbleQueue = append(jc.rumbleQueue, d...)
}

// Perform a SPI read and block for the result.
func (jc *joyconBluetooth) SPIRead(addr uint32, len byte) ([]byte, error) {
	cmd := []byte{0x10, 0, 0, 0, 0, len}
	ch := make(chan []byte)
	chE := make(chan error)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	binary.LittleEndian.PutUint32(cmd[1:], addr)

	jc.mu.Lock()
	jc.subcommandQueue = append(jc.subcommandQueue, cmd)
	jc.spiReads = append(jc.spiReads, spiReadCallback{
		F: func(b []byte, e error) {
			if b != nil {
				ch <- b
			} else {
				chE <- e
			}
		},
		Ctx:     ctx,
		address: addr,
		size:    len,
	})
	jc.mu.Unlock()

	select {
	case data := <-ch:
		return data, nil
	case err := <-chE:
		return nil, err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (jc *joyconBluetooth) SPIWrite(addr uint32, p []byte) error {
	if len(p) > jcpc.SPIMaxData {
		return errors.Errorf("len over maximum")
	}
	cmd := make([]byte, 6+len(p))
	cmd[0] = 0x11
	binary.LittleEndian.PutUint32(cmd[1:], addr)
	cmd[5] = byte(len(p))
	copy(cmd[6:], p)
	jc.queueSubcommand(cmd)
	return nil
}

func (jc *joyconBluetooth) BindToController(c jcpc.Controller) {
	jc.mu.Lock()
	jc.controller = c
	jc.mu.Unlock()

	if c == nil {
		jc.hidHandle.AttemptGrab(false)
	} else {
		jc.hidHandle.AttemptGrab(false)
	}
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
		jc.hidHandle.Close()
	}
	jc.hidHandle = nil
	jc.isShutdown = true
	jc.isAlive = false
	go notify(jc, jcpc.NotifyConnection, jc.ui, jc.controller)
}

func (jc *joyconBluetooth) Reconnect(dev *hid.DeviceInfo) {
	jc.mu.Lock()
	defer jc.mu.Unlock()

	if jc.isShutdown {
		return
	}
	if jc.hidHandle != nil {
		jc.hidHandle.Close()
	}

	handle, err := dev.Device()
	if err != nil {
		fmt.Println("[ ERR] Could not open JoyCon device", err)
		return
	}

	jc.hidHandle = handle
	jc.isAlive = true
	go jc.reader()
	go notify(jc, jcpc.NotifyConnection, jc.ui, jc.controller)
}

func (jc *joyconBluetooth) Close() error {
	jc.mu.Lock()
	defer jc.mu.Unlock()

	if jc.hidHandle != nil {
		jc.hidHandle.Close()
	}
	jc.isAlive = false
	jc.isShutdown = true
	jc.hidHandle = nil
	go notify(jc, jcpc.NotifyConnection, jc.ui, jc.controller)
	return nil
}

func (jc *joyconBluetooth) queueSubcommand(data []byte) {
	jc.mu.Lock()
	jc.subcommandQueue = append(jc.subcommandQueue, data)
	jc.mu.Unlock()
}

// OnFrame triggers writes - this way they're rate-limited
func (jc *joyconBluetooth) OnFrame() {
	connBroken := false
	jc.mu.Lock()
	if !jc.isAlive || jc.isShutdown {
		connBroken = true
	}

	jc.mu.Unlock()

	if connBroken {
		return
	}

	jc.sendRumble(jc.mode.NeedsEmptyRumbles())
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
	handle := jc.hidHandle
	if handle == nil {
		jc.mu.Unlock()
		return
	}
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
	// TODO SetWriteDeadline
	_, err := jc.hidHandle.Write(packet[:])
	if err != nil {
		jc.onReadError(err)
	}
}

func (jc *joyconBluetooth) onReadError(err error) {
	jc.mu.Lock()
	if jc.isShutdown {
		jc.mu.Unlock()
		return // OK
	}
	jc.isAlive = false
	if jc.hidHandle != nil {
		jc.hidHandle.Close()
	}
	jc.hidHandle = nil
	jc.mu.Unlock()

	fmt.Printf("[ ERR] JoyCon %s read error: %v\n", jc.serial, err)

	go notify(jc, jcpc.NotifyConnection, jc.ui, jc.controller)
}

func (jc *joyconBluetooth) fillInput(packet []byte) {
	jc.mu.Lock()

	packet = packet[1:]

	// timer := packet[0]
	prevBattery := jc.packet1 & 0xF0
	jc.packet1 = packet[1]
	if prevBattery != (packet[1] & 0xF0) {
		go jc.ui.JoyConUpdate(jc, jcpc.NotifyBattery)
	}

	jc.buttons = jcpc.ButtonsFromSlice(packet[2:5])

	if jc.side.IsLeft() {
		jc.raw_stick[0][0], jc.raw_stick[0][1] = decodeUint12(packet[5:8])
	}
	if jc.side.IsRight() {
		jc.raw_stick[1][0], jc.raw_stick[1][1] = decodeUint12(packet[8:11])
	}

	jc.mu.Unlock()
}

func gyroDiff(prevFrame, curFrame [6]int16) [6]int16 {
	var result [6]int16
	for j := 0; j < 6; j++ {
		result[j] = curFrame[j] - prevFrame[j]
	}
	return result
}

func gyroPrint(frame [6]int16) {
	fmt.Printf("  %7d %7d %7d %7d %7d %7d\n", frame[0], frame[1], frame[2], frame[3], frame[4], frame[5])
}

func (jc *joyconBluetooth) fillGyroData(packet []byte) {
	if packet[0] != 0x30 {
		return
	}

	jc.mu.Lock()
	defer jc.mu.Unlock()
	if !jc.haveGyro {
		return
	}

	prevFrame := jc.gyro[2]
	for i := 0; i < 3; i++ {
		for j := 0; j < 6; j++ {
			jc.gyro[i][j] = int16(binary.LittleEndian.Uint16(packet[13+2*(i*6+j):]))
		}
	}

	fmt.Print("Gyro data:\n")
	gyroPrint(gyroDiff(prevFrame, jc.gyro[0]))
	gyroPrint(jc.gyro[0])
	gyroPrint(gyroDiff(jc.gyro[0], jc.gyro[1]))
	gyroPrint(jc.gyro[1])
	gyroPrint(gyroDiff(jc.gyro[1], jc.gyro[2]))
	gyroPrint(jc.gyro[2])
}

func (jc *joyconBluetooth) handleSubcommandReply(_packet []byte) {
	packetID := _packet[0]
	packet := _packet[1:]

	replyPacketID := byte(0)
	if packetID == 0x21 {
		replyPacketID = packet[12] - 0x80
	} else /* 0x31-0x33 */ {
		replyPacketID = packet[12]
	}

	if replyPacketID == 0 {
		return
	}

	unknown := false
	switch replyPacketID {
	case 0x01: // Manual Pairing
		unknown = true
	case 0x02: // Device Info
		fmt.Printf("%s: Firmware %d.%d, type %d, MAC %02X:%02X:%02X:%02X:%02X:%02X, colors_on %d\n",
			jc.serial, int(packet[0]), int(packet[1]),
			packet[2], /* packet[3], */
			packet[4], packet[5], packet[6], packet[7], packet[8], packet[9],
			/* packet[10], */
			packet[11],
		)
	case 0x03: // Set Input Report Mode
		unknown = true
	case 0x04: // Shoulder buttons time elapsed
		unknown = true
	case 0x10: // SPI Flash Read
		jc.handleSPIRead(packet[12:])
	case 0x43: // IMU Register Read
		unknown = true
	case 0x50: // Get Voltage
		unknown = true
	case 0x52: // Get unknown data
		unknown = true
	default:
		unknown = true
	}

	if unknown {
		fmt.Printf("got subcommand reply packet: %d\n%s", replyPacketID, hex.Dump(packet[12:]))
	}
}

func (jc *joyconBluetooth) handleButtonPush(packet []byte) {
	if jc.mode != jcpc.InputLazyButtons {
		return
	}

	// translating the buttons is too much of a pain
	// and requires different handling from pro controller
	// so just queue an empty subcommand
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
			time.Sleep(10 * time.Millisecond)
			continue
		}

		n, err := hidHandle.ReadTimeout(buffer[:], 4)
		if err != nil {
			jc.onReadError(err)
			return
		}

		packet := buffer[:n]
		if len(packet) == 0 {
			continue
		}

		switch packet[0] {
		case 0x21:
			jc.fillInput(packet)
			jc.handleSubcommandReply(packet)
			notify(jc, jcpc.NotifyInput, jc.ui, jc.controller)
		case 0x30:
			jc.fillInput(packet)
			jc.fillGyroData(packet)
			notify(jc, jcpc.NotifyInput, jc.ui, jc.controller)
		case 0x31, 0x32, 0x33:
			jc.fillInput(packet)
			jc.handleSubcommandReply(packet)
			notify(jc, jcpc.NotifyInput, jc.ui, jc.controller)
		case 0x3F:
			jc.handleButtonPush(packet)
		default:
			fmt.Println("[!!] Unknown INPUT packet type ", packet[0])
			fmt.Printf("Packet %02X:\n%s", packet[0], hex.Dump(packet[1:]))
		}
	}
}

const (
	factoryStickCalibStart = 0x603D
	factoryStickCalibLen   = 25
	userStickCalibStart    = 0x8010
	userStickCalibLen      = 22
)

func (jc *joyconBluetooth) handleSPIRead(packet []byte) {
	addr := binary.LittleEndian.Uint32(packet[2:])
	length := packet[6]
	data := packet[7:]

	if int(length)+7 <= len(packet) {
		data = packet[7 : 7+length]
	}

	if addr == factoryStickCalibStart && length == factoryStickCalibLen {
		jc.mu.Lock()
		jc.calib[0].Parse(data[0:9], jcpc.TypeLeft)
		jc.calib[1].Parse(data[9:18], jcpc.TypeRight)
		// one padding byte
		const colorOffset = 19
		jc.caseColor.R = data[colorOffset+0]
		jc.caseColor.G = data[colorOffset+1]
		jc.caseColor.B = data[colorOffset+2]
		jc.caseColor.A = 255
		jc.buttonColor.R = data[colorOffset+3]
		jc.buttonColor.G = data[colorOffset+4]
		jc.buttonColor.B = data[colorOffset+5]
		jc.buttonColor.A = 255
		jc.mu.Unlock()

		if true {
			fmt.Printf("%s: Got factory calibration and button colors\n", jc.serial)
		}
		fmt.Printf("%s: SPI read returned [%x+%d]\n%s", jc.serial, addr, length, hex.Dump(data))
	} else if addr == userStickCalibStart && length == userStickCalibLen {
		fmt.Printf("%s: SPI read returned [%x+%d]\n%s", jc.serial, addr, length, hex.Dump(data))
		had := false
		jc.mu.Lock()
		const magicHaveCalibration = 0xA1B2
		if binary.LittleEndian.Uint16(data[0:2]) == magicHaveCalibration {
			jc.calib[0].Parse(data[2:2+9], jcpc.TypeLeft)
			had = true
		}
		if binary.LittleEndian.Uint16(data[11:13]) == magicHaveCalibration {
			jc.calib[1].Parse(data[13:13+9], jcpc.TypeRight)
			had = true
		}
		jc.mu.Unlock()

		if had {
			fmt.Printf("%s: Read user stick calibration: %v\n", jc.serial, jc.calib)
		} else {
			fmt.Printf("%s: Checked user stick calibration: %v\n", jc.serial, jc.calib)
		}
	} else {
		fmt.Printf("%s: SPI read returned [%x+%d]\n%s", jc.serial, addr, length, hex.Dump(data))
	}

	jc.mu.Lock()
	k := 0
	// SliceDeletion
	for i, v := range jc.spiReads {
		if v.address == addr && v.size == length {
			go v.F(data, nil)
		} else {
			if i != k {
				jc.spiReads[k] = v
			}
			k++
		}
	}
	jc.spiReads = jc.spiReads[:k]
	jc.mu.Unlock()
}
