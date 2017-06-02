package consoleiface

import (
	"fmt"
	"os"
	"sort"
	"sync"

	"strings"
	"time"

	"github.com/GeertJohan/go.hid"
	"github.com/riking/joycon/prog4/controller"
	"github.com/riking/joycon/prog4/jcpc"
	"github.com/riking/joycon/prog4/joycon"
)

type outputController struct {
	c  jcpc.Controller
	o  jcpc.Output
	jc []jcpc.JoyCon
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

	// flags to set for the main loop
	doAttemptPairing bool
}

func New(of jcpc.OutputFactory) *Manager {
	return &Manager{
		outputFactory: of,

		commandChan:      make(chan string, 1),
		attemptPairingCh: make(chan struct{}, 1),
	}
}

func (m *Manager) Run() {
	frameTicker := time.NewTicker(16666 * time.Microsecond)
	secondTicker := time.NewTimer(1 * time.Second)
	for {
		select {
		case <-frameTicker.C:
			m.OnFrame()
		case <-secondTicker.C:
			m.SearchDevices()
		case cmd := <-m.commandChan:
			// TODO
			argv := strings.Fields(cmd)
			_ = argv
		case <-m.attemptPairingCh:
			m.attemptPairing()
		}
	}
}

func (m *Manager) OnFrame() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, cv := range m.paired {
		cv.c.OnFrame()
		for _, jc := range cv.jc {
			jc.OnFrame()
		}
	}
	for _, up := range m.unpaired {
		up.jc.OnFrame()
	}
}

var (
	buttonsSLSR_R = jcpc.ButtonState{byte(jcpc.Button_R_SL | jcpc.Button_R_SR), 0, 0}
	buttonsSLSR_L = jcpc.ButtonState{0, 0, byte(jcpc.Button_L_SL | jcpc.Button_L_SR)}
	buttonsRZR    = jcpc.ButtonState{byte(jcpc.Button_R_R | jcpc.Button_R_ZR), 0, 0}
	buttonsLZL    = jcpc.ButtonState{0, 0, byte(jcpc.Button_L_L | jcpc.Button_L_ZL)}

	buttonsAnyLR = jcpc.ButtonState{}.Union(buttonsRZR).Union(buttonsLZL).Union(buttonsSLSR_L).Union(buttonsSLSR_R)
)

func (m *Manager) doPairing(idx1, idx2 int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.doPairing_(idx1, idx2)
}

func (m *Manager) doPairing_(idx1, idx2 int) {
	o, err := m.outputFactory.New(true)
	if err != nil {
		fmt.Println("[FATAL] Failed to create controller output:", err)
		os.Exit(1)
	}
	if idx2 == -1 && m.unpaired[idx1].jc.Type() != jcpc.TypeBoth {
		jc := m.unpaired[idx1].jc
		c := controller.OneJoyCon(jc)
		c.BindToOutput(o)
		m.paired = append(m.paired, outputController{
			c:  c,
			o:  o,
			jc: []jcpc.JoyCon{jc},
		})
	} else if idx2 == -1 {
		jc := m.unpaired[idx1].jc
		c := controller.Pro(jc)
		c.BindToOutput(o)
		m.paired = append(m.paired, outputController{
			c:  c,
			o:  o,
			jc: []jcpc.JoyCon{jc},
		})
	} else {
		jc1 := m.unpaired[idx1].jc
		jc2 := m.unpaired[idx2].jc
		var c jcpc.Controller
		if jc1.Type().IsLeft() {
			c = controller.TwoJoyCons(jc1, jc2)
		} else {
			c = controller.TwoJoyCons(jc2, jc1)
		}
		c.BindToOutput(o)
		m.paired = append(m.paired, outputController{
			c:  c,
			o:  o,
			jc: []jcpc.JoyCon{jc1, jc2},
		})
	}
}

func (m *Manager) attemptPairing() {
	m.mu.Lock()
	defer m.mu.Unlock()
	var didPair []int

	for idx, up := range m.unpaired {
		if up.curButtons.HasAll(buttonsSLSR_L) || up.curButtons.HasAll(buttonsSLSR_R) {
			m.doPairing_(idx, -1)
			didPair = append(didPair, idx)
		} else if up.curButtons.HasAny(buttonsLZL) {
		secondloop:
			for idx2, up2 := range m.unpaired {
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
}

func (m *Manager) JoyConUpdate(jc jcpc.JoyCon) {
	m.mu.Lock()
	defer m.mu.Unlock()

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
			// make a sound using jc.Type()
		}
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
			jc, err = joycon.NewBluetooth(handle, jcpc.TypeLeft)
		case jcpc.JOYCON_PRODUCT_R:
			jc, err = joycon.NewBluetooth(handle, jcpc.TypeRight)
		case jcpc.JOYCON_PRODUCT_PRO:
			jc, err = joycon.NewBluetooth(handle, jcpc.TypeBoth)
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
				continue
			}
			jc, err = joycon.NewChargeGrip(handle, handleR)
			if err != nil {
				handleR.Close()
			}
		}
		if err != nil {
			handle.Close()
		}

		m.unpaired = append(m.unpaired, unpairedController{jc: jc})
	} // range deviceList
	return nil
}
