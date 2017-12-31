package consoleiface

import (
	"fmt"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/GeertJohan/go.hid"
	"github.com/riking/joycon/prog4/controller"
	"github.com/riking/joycon/prog4/jcpc"
	"github.com/riking/joycon/prog4/joycon"
)

const maxControllerCount = 4

type outputController struct {
	pNum int
	c    jcpc.Controller
	o    jcpc.Output
	jc   []jcpc.JoyCon
}

type unpairedController struct {
	jc          jcpc.JoyCon
	curButtons  jcpc.ButtonState
	prevButtons jcpc.ButtonState
}

type Manager struct {
	mu       sync.Mutex
	paired   []outputController
	unpaired []unpairedController

	wantReconnect []jcpc.JoyCon

	outputFactory jcpc.OutputFactory

	commandChan      chan string
	attemptPairingCh chan struct{}
	consoleExit      chan struct{}

	// flags to set for the main loop
	doAttemptPairing bool
}

func New(of jcpc.OutputFactory) *Manager {
	m := &Manager{
		outputFactory: of,

		commandChan:      make(chan string, 1),
		attemptPairingCh: make(chan struct{}, 1),
		consoleExit:      make(chan struct{}),
	}

	return m
}

func (m *Manager) Run() {
	frameTicker := time.NewTicker(16666 * time.Microsecond)
	secondTicker := time.NewTicker(1 * time.Second)

	go m.readStdin()

	for {
		select {
		case <-frameTicker.C:
			m.OnFrame()
		case <-secondTicker.C:
			m.SearchDevices()
		case <-m.attemptPairingCh:
			m.attemptPairing()

		case <-m.consoleExit:
			fmt.Println("Disconnecting controllers...")
			for _, cv := range m.paired {
				cv.c.Close()
				cv.o.Close()
				for _, jc := range cv.jc {
					//jc.Shutdown()
					jc.Close()
				}
			}
			for _, up := range m.unpaired {
				//up.jc.Shutdown()
				up.jc.Close()
			}
			time.Sleep(200 * time.Millisecond)
			return
		}
	}
}

func (m *Manager) OnFrame() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, cv := range m.paired {
		cv.o.OnFrame()
		cv.c.OnFrame()
		for _, jc := range cv.jc {
			jc.OnFrame()
		}
	}
	for _, up := range m.unpaired {
		up.jc.OnFrame()
	}
}

// must be called locked
func (m *Manager) assignPlayerNumber() int {
	var used [maxControllerCount]bool

	for _, v := range m.paired {
		used[v.pNum-1] = true
	}
	for i := 0; i < maxControllerCount; i++ {
		if !used[i] {
			return i + 1
		}
	}
	return 0
}

var (
	buttonsSLSR_R = jcpc.ButtonState{byte((jcpc.Button_R_SL | jcpc.Button_R_SR) & 0xFF), 0, 0}
	buttonsSLSR_L = jcpc.ButtonState{0, 0, byte((jcpc.Button_L_SL | jcpc.Button_L_SR) & 0xFF)}
	buttonsRZR    = jcpc.ButtonState{byte((jcpc.Button_R_R | jcpc.Button_R_ZR) & 0xFF), 0, 0}
	buttonsLZL    = jcpc.ButtonState{0, 0, byte((jcpc.Button_L_L | jcpc.Button_L_ZL) & 0xFF)}
	buttonsLR     = jcpc.ButtonState{byte(jcpc.Button_R_R & 0xFF), 0, byte(jcpc.Button_L_L & 0xFF)}
	buttonsZLZR   = jcpc.ButtonState{byte(jcpc.Button_R_ZR & 0xFF), 0, byte(jcpc.Button_L_ZL & 0xFF)}

	buttonsAnyLR = jcpc.ButtonState{}.Union(buttonsRZR).Union(buttonsLZL).Union(buttonsSLSR_L).Union(buttonsSLSR_R)
)

func (m *Manager) doPairing(idx1, idx2 int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.doPairing_(idx1, idx2)

	didPair := []int{idx1, idx2}
	if idx2 == -1 {
		didPair = []int{idx1}
	}
	newUnpaired := make([]unpairedController, len(m.unpaired)-len(didPair))
	sort.Ints(didPair)
	i := 0
	k := 0
	for j, v := range m.unpaired {
		if j == didPair[k] {
			k++
		} else {
			newUnpaired[i] = v
			i++
		}
	}
}

func (m *Manager) doPairing_(idx1, idx2 int) {
	pNum := m.assignPlayerNumber()

	if idx2 == -1 && m.unpaired[idx1].jc.Type() != jcpc.TypeBoth {
		fmt.Println("pairing single")
		jc := m.unpaired[idx1].jc
		o, err := m.outputFactory(jc.Type(), pNum)
		if err != nil {
			fmt.Println("[FATAL] Failed to create controller output:", err)
			os.Exit(1)
		}
		c := controller.OneJoyCon(jc, m)
		c.BindToOutput(o)
		jc.BindToController(c)
		m.paired = append(m.paired, outputController{
			c:    c,
			o:    o,
			jc:   []jcpc.JoyCon{jc},
			pNum: pNum,
		})
	} else if idx2 == -1 {
		fmt.Println("pairing pro")
		jc := m.unpaired[idx1].jc
		o, err := m.outputFactory(jcpc.TypeBoth, pNum)
		if err != nil {
			fmt.Println("[FATAL] Failed to create controller output:", err)
			os.Exit(1)
		}
		c := controller.Pro(jc, m)
		c.BindToOutput(o)
		jc.BindToController(c)
		m.paired = append(m.paired, outputController{
			c:    c,
			o:    o,
			jc:   []jcpc.JoyCon{jc},
			pNum: pNum,
		})
	} else {
		fmt.Println("pairing double")
		jc1 := m.unpaired[idx1].jc
		jc2 := m.unpaired[idx2].jc
		o, err := m.outputFactory(jcpc.TypeBoth, pNum)
		if err != nil {
			fmt.Println("[FATAL] Failed to create controller output:", err)
			os.Exit(1)
		}
		var c jcpc.Controller
		if jc1.Type().IsLeft() {
			c = controller.TwoJoyCons(jc1, jc2, m)
		} else {
			c = controller.TwoJoyCons(jc2, jc1, m)
		}
		c.BindToOutput(o)
		jc1.BindToController(c)
		jc2.BindToController(c)
		m.paired = append(m.paired, outputController{
			c:    c,
			o:    o,
			jc:   []jcpc.JoyCon{jc1, jc2},
			pNum: pNum,
		})
	}
	m.fixPlayerLights()
}

var playerLightSeq = []byte{0xF0, 0x01, 0x03, 0x07, 0x0F, 0x05, 0x09, 0x06, 0x0A}

func (m *Manager) fixPlayerLights() {
	// TODO separate business logic and moving arrays around
	// Fix player lights
	for _, c := range m.paired {
		for _, jc := range c.jc {
			jcpc.SetPlayerLights(jc, playerLightSeq[c.pNum])
		}
	}

	for i, up := range m.unpaired {
		jcpc.SetPlayerLights(up.jc, byte((i+1)<<4))
	}
}

func (m *Manager) removeFromUnpaired_Locked(idx int) {
	m.unpaired = append(m.unpaired[:idx], m.unpaired[idx+1:]...)
}

func (m *Manager) attemptPairing() {
	m.mu.Lock()
	defer m.mu.Unlock()
	var didPair []int

	for idx, up := range m.unpaired {
		if up.curButtons.HasAll(buttonsSLSR_L) || up.curButtons.HasAll(buttonsSLSR_R) {
			m.doPairing_(idx, -1)
			didPair = append(didPair, idx)
		} else if up.curButtons.HasAll(buttonsLR) || up.curButtons.HasAll(buttonsZLZR) {
			m.doPairing_(idx, -1)
			didPair = append(didPair, idx)
		} else if up.curButtons.HasAny(buttonsLZL) {
		secondloop:
			for idx2, up2 := range m.unpaired {
				if idx == idx2 {
					continue
				}
				for _, usedIdx := range didPair {
					if idx2 == usedIdx {
						continue secondloop
					}
				}
				if up2.curButtons.HasAny(buttonsRZR) {
					m.doPairing_(idx, idx2)
					didPair = append(didPair, idx, idx2)
					break
				}
			}
		}
	}

	if len(didPair) > 0 {
		sort.Ints(didPair)
		if len(didPair) > 1 {
			m.removeFromUnpaired_Locked(didPair[1])
		}
		m.removeFromUnpaired_Locked(didPair[0])
	}
}

func (m *Manager) JoyConUpdate(jc jcpc.JoyCon, flags int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if flags&jcpc.NotifyConnection != 0 {
		idx := -1
		for i, aJC := range m.wantReconnect {
			if jc == aJC {
				idx = i
				break
			}
		}
		if !jc.WantsReconnect() != (idx == -1) {
			if idx == -1 {
				m.wantReconnect = append(m.wantReconnect, jc)
				fmt.Printf("JoyCon %s needs reconnecting\n", jc.Serial())
			} else {
				m.wantReconnect = append(m.wantReconnect[:idx], m.wantReconnect[idx+1:]...)
			}
		}

		idx = -1
		for i, v := range m.unpaired {
			if jc == v.jc {
				idx = i
				break
			}
		}
		if idx != -1 {
			if jc.IsStopping() {
				fmt.Println("removing", jc.Serial())
				m.removeFromUnpaired_Locked(idx)
				return
			}
		}
	}

	if flags&jcpc.NotifyInput != 0 {
		idx := -1
		for i, v := range m.unpaired {
			if jc == v.jc {
				idx = i
				break
			}
		}
		if idx != -1 {
			up := &m.unpaired[idx]
			up.prevButtons = up.curButtons
			up.curButtons = jc.Buttons()
			diff := up.curButtons.DiffMask(up.prevButtons)
			if diff.HasAny(buttonsAnyLR) {
				select {
				case m.attemptPairingCh <- struct{}{}:
				default:
				}
			}
			if up.curButtons.HasAny(diff) {
				fmt.Println("plonk")
				// make a sound on the ui?
			}
		}
	}

	if flags&jcpc.NotifyBattery != 0 {
		fmt.Printf("%s (%s): %s\n", jc.Type().String(), jc.Serial(), renderBattery(jc.Battery()))
	}
}

func (m *Manager) SearchDevices() error {
	deviceList, err := hid.Enumerate(jcpc.VENDOR_NINTENDO, 0)
	if err != nil {
		fmt.Println("Enumeration error:", err)
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

outer:
	for i, dev := range deviceList {
		// Check for reconnects
		for _, jc := range m.wantReconnect {
			if jc.Serial() == dev.SerialNumber && jc.WantsReconnect() {
				jc.Reconnect(dev)
				continue outer
			}
		}

		// Check for already existing
		for _, upV := range m.unpaired {
			if upV.jc.Serial() == dev.SerialNumber {
				continue outer
			}
		}
		for _, c := range m.paired {
			for _, jc := range c.jc {
				if jc.Serial() == dev.SerialNumber {
					continue outer
				}
			}
		}

		handle, err := dev.Device()
		if err != nil {
			if os.IsPermission(err) {
				fmt.Println("Couldn't open JoyCon device - install udev rules or run as sudo")
			} else {
				fmt.Println("Couldn't open JoyCon device:", err)
			}
			return err
		}
		var jc jcpc.JoyCon
		switch dev.ProductId {
		case jcpc.JOYCON_PRODUCT_L:
			jc, err = joycon.NewBluetooth(handle, jcpc.TypeLeft, m)
		case jcpc.JOYCON_PRODUCT_R:
			jc, err = joycon.NewBluetooth(handle, jcpc.TypeRight, m)
		case jcpc.JOYCON_PRODUCT_PRO:
			jc, err = joycon.NewBluetooth(handle, jcpc.TypeBoth, m)
		case jcpc.JOYCON_PRODUCT_CHARGEGRIP:
			if dev.InterfaceNumber == 1 {
				handle.Close()
				continue
			}
			var handleR *hid.Device
			for _, dev2 := range deviceList[i:] {
				if dev2.InterfaceNumber == 1 && dev2.ProductId == jcpc.JOYCON_PRODUCT_CHARGEGRIP {
					handleR, err = dev2.Device()
					break
				}
			}
			if err != nil {
				fmt.Println("Couldn't open JoyCon device:", err)
				handle.Close()
				return err
			}
			if handleR == nil {
				// must have both to be recognized
				handle.Close()
				break
			}
			jc, err = joycon.NewChargeGrip(handle, handleR)
			if err != nil {
				handleR.Close()
				break
			}
		}
		if err != nil {
			handle.Close()
			fmt.Println("Couldn't initialize JoyCon:", err)
			continue outer
		}

		m.unpaired = append(m.unpaired, unpairedController{jc: jc})
		fmt.Println("[INFO] Connected to", jc.Type(), jc.Serial())
	} // range deviceList
	m.fixPlayerLights()
	return nil
}

func (m *Manager) RemoveController(c jcpc.Controller) {
	m.mu.Lock()
	defer m.mu.Unlock()

	idx := -1
	for i, v := range m.paired {
		if v.c == c {
			idx = i
			break
		}
	}
	if idx == -1 {
		return
	}
	v := m.paired[idx]
	m.paired = append(m.paired[:idx], m.paired[idx+1:]...)
	v.c.Close()
	v.o.Close()
	for _, jc := range v.jc {
		jc.Close()
	}
}
