//+build !linux linux,nobluez

package main

import (
	"time"

	"github.com/riking/joycon/prog4/jcpc"
)

func getBluetoothManager() (jcpc.BluetoothManager, error) {
	d := &dummyBTManager{}
	d.chOut = make(chan jcpc.BluetoothDeviceNotification, 10)
	d.tickerOn = false

	return d, nil
}

// dummyBTManager does nothing and defers to the OS stack's native user
// interface.
type dummyBTManager struct {
	chOut chan jcpc.BluetoothDeviceNotification

	mu         sync.Mutex
	ticker     *time.Ticker
	tickerOn   bool
	tickerStop chan struct{}
}

func (m *dummyBTManager) SavePairingInfo(mac [6]byte) {}
func (m *dummyBTManager) DeletePairingInfo()          {}

// InitialScan emits an empty notification.
func (m *dummyBTManager) InitialScan() {
	m.chOut <- jcpc.BluetoothDeviceNotification{}
}

// StartDiscovery emits empty notifications every 2 seconds.
func (m *dummyBTManager) StartDiscovery() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.tickerOn {
		m.tickerOn = true
		m.ticker = time.NewTicker(2 * time.Second)
		m.tickerStop = make(chan struct{})
		go m.tickerNotify()
	}
}

// StopDiscovery stops the every-five-second notifications.
func (m *dummyBTManager) StopDiscovery() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.tickerOn {
		m.tickerOn = false
		m.ticker.Stop()
		close(m.tickerStop)
	}
}

// NotifyChannel implements BluetoothManager.
func (m *dummyBTManager) NotifyChannel() <-chan jcpc.BluetoothDeviceNotification {
	return m.chOut
}

func (m *dummyBTManager) tickerNotify() {
	for {
		select {
		case <-m.ticker.C:
			m.chOut <- jcpc.BluetoothDeviceNotification{}
		case <-m.tickerStop:
			return
		}
	}
}
