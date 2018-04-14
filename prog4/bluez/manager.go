package bluez

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/godbus/dbus"
	"github.com/godbus/dbus/introspect"
	"github.com/pkg/errors"
	"github.com/riking/joycon/prog4/jcpc"
)

// JoyconAPI presents a manageable surface area for the rest of the code to
// use.  Eventually it will be turned into an interface for multi-OS
// functionality.
type JoyconAPI struct {
	// write-once
	changeNotify chan jcpc.BluetoothDeviceNotification
	busConn      *dbus.Conn
	busSignalCh  chan *dbus.Signal

	// protected by mu
	mu               sync.Mutex
	discoveryEnabled bool
	adapterPaths     []dbus.ObjectPath
	devicePaths      map[dbus.ObjectPath]btDeviceInfo

	setupDone         bool
	setupChangeBuffer []*dbus.Signal
}

type dbusObjectNotify map[string]map[string]dbus.Variant

// initial setup flow:
//
// API created
//   connect to dbus
//   set up signal listener
// InitialScan()
//   subscribe to change signals
//     buffer actual changes
//   call GetManagedObjects
//   set up btDeviceInfo structs (several mutex lock-unlocks)
//   locked: set setupDone, process buffered changes

type btDeviceInfo struct {
	Name string
	MAC  [6]byte

	Paired        bool
	Trusted       bool
	Connected     bool
	IsInputDevice bool
	IsJoyCon      bool
}

var _ jcpc.BluetoothManager = &JoyconAPI{}

func New() (*JoyconAPI, error) {
	busConn, err := dbus.SystemBus()
	if err != nil {
		return nil, err
	}
	signalCh := make(chan *dbus.Signal, 16)
	busConn.Signal(signalCh)
	a := &JoyconAPI{
		changeNotify: make(chan jcpc.BluetoothDeviceNotification, 16),
		busConn:      busConn,
		busSignalCh:  signalCh,
	}
	go a.handleChangeSignals()
	return a, nil
}

// Request discovery of Bluetooth devices (e.g., entered the "change controller
// config" screen).
func (a *JoyconAPI) StartDiscovery() {
	err := a.startDiscovery()
	if err != nil {
		fmt.Fprintf(os.Stderr, "\r[bluez] failed to start bluetooth discovery: %v\n", err)
	}
}

func (a *JoyconAPI) startDiscovery() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.discoveryEnabled {
		a.discoveryEnabled = true
		// TODO request discovery...
	}
	return nil
}

// Stop automatic discovery of Bluetooth devices.
func (a *JoyconAPI) StopDiscovery() {
	err := a.stopDiscovery()
	if err != nil {
		fmt.Fprintf(os.Stderr, "\r[bluez] failed to stop bluetooth discovery: %v\n", err)
	}
}

func (a *JoyconAPI) stopDiscovery() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.discoveryEnabled {
		a.discoveryEnabled = false
		// TODO request stop discovery...
	}
	return nil
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
	err := a.deletePairingInfo()
	if err != nil {
		fmt.Fprintf(os.Stderr, "\r[bluez] failed to delete pairing info: %v\n", err)
	}
}

func (a *JoyconAPI) deletePairingInfo() error {
	// listBluetoothDevices()
	// for dev := devices
	// if isJoyCon(dev)
	// getAdapter(dev.adapter).RemoveDevice(path)
	return nil
}

func (a *JoyconAPI) SavePairingInfo(mac [6]byte) {
	macStr := fmt.Sprintf("%02X_%02X_%02X_%02X_%02X_%02X",
		mac[0], mac[1], mac[2], mac[3], mac[4], mac[5],
	)
	err := a.savePairingInfo(macStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\r[bluez] failed to save bluetooth pairing info for %s: %v\n", macStr, err)
	}
}

func (a *JoyconAPI) savePairingInfo(macStr string) error {
	// set Trusted=true
	for k := range a.devicePaths {
		if strings.HasSuffix(string(k), macStr) {
			obj := a.busConn.Object(BlueZBusName, k)
			c := obj.Call("org.freedesktop.DBus.Properties.Set", 0, "org.bluez.Device1", "Trusted", true)
			return c.Err
		}
	}
	return errors.New("device not found")
}

func (a *JoyconAPI) handleChangeSignals() {
	for busSig := range a.busSignalCh {
		fmt.Println("[bluez]", "dbus signal", busSig)
		if busSig.Name == "org.freedesktop.DBus.ObjectManager.InterfacesAdded" {
		} else if busSig.Name == "org.freedesktop.DBus.ObjectManager.InterfacesRemoved" {
		}
	}
}

func (a *JoyconAPI) InitialScan() {
	err := a.initialScan()
	if err != nil {
		fmt.Fprintf(os.Stderr, "\r[bluez] failed to check bluetooth devices: %v\n", err)
	}
}

func (a *JoyconAPI) initialScan() error {
	fmt.Println("Starting initial scan")
	// Subscribe to InterfaceAdded/InterfaceRemoved
	busObj := a.busConn.BusObject()
	sigCall := busObj.Go("org.freedesktop.DBus.AddMatch", 0, nil,
		"type='signal',sender='org.bluez',interface='org.freedesktop.DBus.ObjectManager',path='/'")
	if sigCall.Err != nil {
		return errors.Wrap(sigCall.Err, "subscribe to updates")
	}
	<-sigCall.Done
	if sigCall.Err != nil {
		return errors.Wrap(sigCall.Err, "subscribe to updates")
	}
	fmt.Println("done with addmatchsignal")

	// Call GetManagedObjects
	obj := a.busConn.Object(BlueZBusName, "/")
	call := obj.Call("org.freedesktop.DBus.ObjectManager.GetManagedObjects", 0)
	if call.Err != nil {
		return errors.Wrap(call.Err, "get current objects")
	}

	var managedObjectsReturn map[dbus.ObjectPath]dbusObjectNotify
	err := call.Store(&managedObjectsReturn)
	if err != nil {
		return errors.Wrap(err, "get current objects")
	}

	for path, v := range managedObjectsReturn {
		a.checkDBusNewObject(path, v)
	}

	a.mu.Lock()
	a.setupDone = true
	a.mu.Unlock()

	return nil
	/*
		ispectNode, err := introspect.Call(obj)
		if err != nil {
			return errors.Wrapf(err, "introspect %s", BlueZRootPath)
		}

		var adapterList []dbus.ObjectPath
		for _, v := range ispectNode.Children {
			adapterList = append(adapterList, joinPath(path, v.Name))
		}

		a.mu.Lock()
		a.adapterPaths = adapterList
		a.mu.Unlock()

		fmt.Println("[bluez] adapter check: found", len(adapterList))
		for _, v := range adapterList {
			err = a.checkAdapter(v)
			if err != nil {
				fmt.Fprintf(os.Stderr, "\r[bluez] failed to check bluetooth devices under %s: %v\n", v, err)
			}
		}
		return nil
	*/
}

func (a *JoyconAPI) checkDBusNewObject(path dbus.ObjectPath, data dbusObjectNotify) error {
	_, ok := data[Adapter1Interface]
	if ok {
		a.mu.Lock()
		found := false
		for _, v := range a.adapterPaths {
			if v == path {
				found = true
				break
			}
		}
		if !found {
			a.adapterPaths = append(a.adapterPaths, path)
		}
		a.mu.Unlock()
		fmt.Println("[bluez] found adapter", path)
	}
	deviceData, ok := data[Device1Interface]
	if ok {
		fmt.Println("[bluez] found device", path, deviceData)
		// TODO
	}
	return nil
}

func (a *JoyconAPI) processRemoval(path dbus.ObjectPath, ifaces []string) {
	for _, iface := range ifaces {
		if iface == Adapter1Interface {
			a.mu.Lock()
			idx := -1
			for i, v := range a.adapterPaths {
				if v == path {
					idx = i
					break
				}
			}
			if idx != -1 {
				copy(a.adapterPaths[idx:], a.adapterPaths[idx+1:])
				a.adapterPaths = a.adapterPaths[:len(a.adapterPaths)-1]
			}
			a.mu.Unlock()
		}
		if iface == Device1Interface {
			a.mu.Lock()
			devInfo := a.devicePaths[path]
			if devInfo.IsJoyCon {
				a.emitNotify(path, false, false)
			}
			a.mu.Unlock()
		}
	}
}

func (a *JoyconAPI) emitNotify(path dbus.ObjectPath, connected, newDevice bool) {
	var notify jcpc.BluetoothDeviceNotification
	if !parseMACPath(&notify, path) {
		fmt.Println("[bluez] [ERR] could not parse device MAC", path)
		return
	}
	notify.Connected = connected
	notify.NewDevice = newDevice
	a.changeNotify <- notify
}

func parseMACPath(dest *jcpc.BluetoothDeviceNotification, path dbus.ObjectPath) bool {
	idx := strings.LastIndex(string(path), "/")
	if idx < 0 {
		return false
	}
	macSplit := strings.Split(string(path)[idx:], "_")
	if len(macSplit) != 7 {
		return false
	}
	for i := 0; i < 6; i++ {
		by, err := strconv.ParseUint(macSplit[i+1], 16, 8)
		if err != nil {
			return false
		}
		dest.MAC[i] = byte(by)
	}
	dest.MACString = fmt.Sprintf("%02X:%02X:%02X:%02X:%02X:%02X",
		dest.MAC[0],
		dest.MAC[1],
		dest.MAC[2],
		dest.MAC[3],
		dest.MAC[4],
		dest.MAC[5],
	)
	return true
}

func (a *JoyconAPI) checkAdapter(path dbus.ObjectPath, data dbusObjectNotify) error {
	obj := a.busConn.Object(BlueZBusName, path)
	var adapterAddrV dbus.Variant
	err := obj.Call("org.freedesktop.DBus.Properties.Get", 0, Adapter1Interface, "Address").Store(&adapterAddrV)
	if err != nil {
		return errors.Wrap(err, "get .Address")
	}
	if adapterAddr, ok := adapterAddrV.Value().(string); ok {
		// check adapter vs blacklist
		_ = adapterAddr
		fmt.Println("[bluez]", "adapter check:", "found", path, adapterAddr)
	} else {
		fmt.Printf("[bluez] adapter check: addr not a string, got %T %v\n", adapterAddrV.Value(), adapterAddrV.Value())
	}

	fmt.Println("[bluez] introspect", obj)
	ispectNode, err := introspect.Call(obj)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\r[dbus] failed to check bluetooth devices: introspect %s: %v\n", path, err)
		return err
	}
	fmt.Println("[bluez] introspect found", len(ispectNode.Children), "device records")
	for _, v := range ispectNode.Children {
		a.checkDevice(joinPath(path, v.Name))
	}
	return nil
}

func (a *JoyconAPI) checkDevice(path dbus.ObjectPath) {
	// a.mu.Lock()
	// prevInfo := a.devicePaths[path]
	// a.mu.Unlock()

	info, err := a.getDeviceInfo(path)
	if err != nil {
		fmt.Println("[dbus]", "device", path, "error", err)
	} else {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		fmt.Println("[dbus]", "device", path, "info")
		enc.Encode(info)
	}
}

func (a *JoyconAPI) getDeviceInfo(path dbus.ObjectPath) (btDeviceInfo, error) {
	var newInfo btDeviceInfo
	obj := a.busConn.Object(BlueZBusName, path)

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
			return newInfo, errors.Wrapf(c.Err, "get device info for %s", path)
		}
	}

	var err error
	var macStr string
	var uuids []string
	err = calls[0].Store(&newInfo.Name)
	if err != nil {
		return newInfo, errors.Wrapf(err, "read device info for %s", path)
	}
	err = calls[1].Store(&macStr)
	if err != nil {
		return newInfo, errors.Wrapf(err, "read device info for %s", path)
	}
	err = calls[2].Store(&newInfo.Paired)
	if err != nil {
		return newInfo, errors.Wrapf(err, "read device info for %s", path)
	}
	err = calls[3].Store(&newInfo.Trusted)
	if err != nil {
		return newInfo, errors.Wrapf(err, "read device info for %s", path)
	}
	err = calls[4].Store(&newInfo.Connected)
	if err != nil {
		return newInfo, errors.Wrapf(err, "read device info for %s", path)
	}
	err = calls[5].Store(&uuids)
	if err != nil {
		return newInfo, errors.Wrapf(err, "read device info for %s", path)
	}

	macSplit := strings.Split(macStr, ":")
	if len(macSplit) != 6 {
		return newInfo, errors.Wrapf(errors.Errorf("wrong number of colons in MAC address"),
			"get device info for %s", path)
	}
	for i := 0; i < 6; i++ {
		by, err := strconv.ParseUint(macSplit[i], 16, 8)
		if err != nil {
			return newInfo, errors.Wrapf(err, "get device info for %s: bad MAC address", path)
		}
		newInfo.MAC[i] = byte(by)
	}
	for _, v := range uuids {
		if v == HIDProfileUUIDStrL || v == HIDProfileUUIDStrU {
			newInfo.IsInputDevice = true
			break
		}
	}
	for _, jcStr := range autoconnectDeviceNames {
		if newInfo.Name == jcStr {
			newInfo.IsJoyCon = true
			break
		}
	}
	return newInfo, nil
}

func joinPath(parent dbus.ObjectPath, child string) dbus.ObjectPath {
	return dbus.ObjectPath(string(parent) + "/" + child)
}
