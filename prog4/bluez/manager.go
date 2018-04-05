package bluez

import (
	"github.com/godbus/dbus"
)

// JoyconAPI presents a manageable surface area for the rest of the code to
// use.  Eventually it will be turned into an interface for multi-OS
// functionality.
type JoyconAPI struct {
	mu sync.Mutex

	discoveryEnabled bool
	changeNotify     chan jcpc.BluetoothDeviceNotification

	busConn      *dbus.Conn
	busSignalCh  chan *dbus.Signal
	adapterPaths []dbus.ObjectPath
	devicePaths  map[dbus.ObjectPath]prevDeviceInfo
}

type prevDeviceInfo struct {
	Name string
	MAC  [6]byte

	Paired        bool
	Trusted       bool
	Connected     bool
	IsInputDevice bool
}

var _ jcpc.BluetoothManager = &JoyconAPI{}

func New() (*JoyconAPI, error) {
	busConn, err := dbus.SystemBus()
	if err != nil {
		return nil, err
	}
	signalCh := make(chan *dbus.Signal, 16)
	busConn.Signal(signalCh)
	jc := &JoyconAPI{
		changeNotify: make(chan jcpc.BluetoothDeviceNotification, 16),
		busConn:      busConn,
		busSignalCh:  signalCh,
		objectPaths:  nil,
	}
	go jc.handleSignals()
	return jc, nil
}

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
	str := fmt.Sprintf("/dev_%02X_%02X_%02X_%02X_%02X_%02X",
		mac[0], mac[1], mac[2], mac[3], mac[4], mac[5],
	)

}

func (a *JoyconAPI) InitialScan() {
	ispectNode, err := introspect.Call(a.busConn.Object(BlueZBusName, BlueZRootPath))
	if err != nil {
		fmt.Fprintf(os.Stderr, "\r[dbus] failed to check bluetooth devices: introspect %s: %v\n", BlueZRootPath, err)
		return
	}

	var adapterList []string
	for _, v := range ispectNode.Children {
		adapterList = append(adapterList, v.Name)
	}

	a.mu.Lock()
	a.adapterPaths = adapterList
	a.mu.Unlock()

	for _, v := range ispectNode.Children {
		a.checkAdapter(v.Name)
	}
}

func (a *JoyconAPI) checkAdapter(path string) {
	obj := a.busConn.Object(BlueZBusName, path)
	adapterAddrV, err := obj.GetProperty(Adapter1Interface + ".Address")
	if err != nil {
		fmt.Fprintf(os.Stderr, "\r[dbus] failed to check bluetooth devices: get .Address %s: %v\n", path, err)
		return
	}
	if adapterAddr, ok := adapterAddrV.Value().(string); ok {
		// check adapter vs blacklist
	}

	ispectNode, err := introspect.Call(obj)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\r[dbus] failed to check bluetooth devices: introspect %s: %v\n", path, err)
		return
	}
	for _, v := range ispectNode.Children {
		a.checkDevice(v.Name)
	}
}

func (a *JoyconAPI) checkDevice(path string) {
	obj := a.busConn.Object(BlueZBusName, path)
	a.mu.Lock()
	prevInfo := a.devicePaths[path]
	a.mu.Unlock()

	// name, MAC, paired, trusted, connected, UUIDs
	var calls = [...]*dbus.Call{
		obj.Go("org.freedesktop.DBus.Properties.Get", 0, nil, Device1Interface, "Name"),
		obj.Go("org.freedesktop.DBus.Properties.Get", 0, nil, Device1Interface, "Address"),
		obj.Go("org.freedesktop.DBus.Properties.Get", 0, nil, Device1Interface, "Paired"),
		obj.Go("org.freedesktop.DBus.Properties.Get", 0, nil, Device1Interface, "Trusted"),
		obj.Go("org.freedesktop.DBus.Properties.Get", 0, nil, Device1Interface, "Connected"),
		obj.Go("org.freedesktop.DBus.Properties.Get", 0, nil, Device1Interface, "UUIDs"),
	}
	for _, c := range calls[:] {
		<-c.Done
		if c.Err != nil {
			fmt.Fprintf(os.Stderr, "\r[dbus] error Get Property %s: %v\n", path, c.Err)
			return
		}
	}
	var newInfo prevDeviceInfo
	calls[0].Store(&newInfo.Name)
	var macStr string
	calls[1].Store(&macStr)
	calls[2].Store(&newInfo.Paired)
	calls[3].Store(&newInfo.Trusted)
	calls[4].Store(&newInfo.Connected)
	var uuids []string
	calls[5].Store(&newInfo.UUIDs)
}
