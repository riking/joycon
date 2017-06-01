package consoleiface

import (
	"fmt"
	"os"

	"github.com/GeertJohan/go.hid"
	"github.com/riking/joycon/prog4/jcpc"
	"github.com/riking/joycon/prog4/joycon"
)

type outputController struct {
	c  jcpc.Controller
	o  jcpc.Output
	jc []jcpc.JoyCon
}

type Manager struct {
	paired   []outputController
	unpaired []jcpc.JoyCon

	wantReconnect []jcpc.JoyCon
}

func (m *Manager) SearchDevices() error {
	deviceList, err := hid.Enumerate(jcpc.VENDOR_NINTENDO, 0)
	if err != nil {
		fmt.Println("Enumeration error:", err)
		return err
	}
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
			if upV.Serial() == dev.SerialNumber {
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
			jc, err = joycon.NewBluetooth(handle, jcpc.JCSideLeft)
		case jcpc.JOYCON_PRODUCT_R:
			jc, err = joycon.NewBluetooth(handle, jcpc.JCSideRight)
		case jcpc.JOYCON_PRODUCT_PRO:
			jc, err = joycon.NewBluetooth(handle, jcpc.JCSideBoth)
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

		m.unpaired = append(m.unpaired, jc)
	} // range deviceList
	return nil
}
