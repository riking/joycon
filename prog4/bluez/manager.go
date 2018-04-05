package bluez

// JoyconAPI presents a manageable surface area for the rest of the code to
// use.  Eventually it will be turned into an interface for multi-OS
// functionality.
type JoyconAPI struct {
	mu sync.Mutex

	discoveryEnabled bool
	changeNotify     chan jcpc.BluetoothDeviceNotification
}

var _ jcpc.BluetoothManager = &JoyconAPI{}

// Request discovery of Bluetooth devices (e.g., entered the "change controller
// config" screen).
func (a *JoyconAPI) StartDiscovery() {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.discoveryEnabled {
		a.discoveryEnabled = true
		// TODO request discovery...
	}
}

// Stop automatic discovery of Bluetooth devices.
func (a *JoyconAPI) StopDiscovery() {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.discoveryEnabled {
		a.discoveryEnabled = false
		// TODO request stop discovery...
	}
}

// Returns whether the manager object thinks Bluetooth discovery is enabled.
func (a *JoyconAPI) IsDiscoveryEnabled() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.discoveryEnabled
}

func (a *JoyconAPI) NotifyChannel() <-chan jcpc.BluetoothDeviceNotification {
	return a.changeNotify
}

// Call this when the user holds the device sync button down.
func (a *JoyconAPI) DeletePairingInfo() {
	// listBluetoothDevices()
	// for dev := devices
	// if isJoyCon(dev)
	// getAdapter(dev.adapter).RemoveDevice(path)
}

func (a *JoyconAPI) SavePairingInfo(mac [6]byte) {
	// set Trusted=true
}
