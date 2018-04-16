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
	MAC  [6]byte `json:"-"`

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
		devicePaths:  make(map[dbus.ObjectPath]btDeviceInfo),
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
	//wasEnabled := a.discoveryEnabled
	if !a.discoveryEnabled {
		a.discoveryEnabled = true
	}
	adapterList := a.adapterPaths
	a.mu.Unlock()

	fmt.Println("[bluez] starting bt discovery")
	// Request discovery from every adapter
	ch := make(chan *dbus.Call, len(adapterList))
	for _, path := range adapterList {
		a.busConn.Object(BlueZBusName, path).Go("org.bluez.Adapter1.StartDiscovery", 0, ch)
	}
	for _ = range adapterList {
		call := <-ch
		if call.Err != nil {
			fmt.Fprintf(os.Stderr,
				"\r[bluez] failed to start bluetooth discovery for %s: %v\n",
				strings.TrimPrefix(string(call.Path), "/org/bluez/"),
				call.Err,
			)
		}
	}
	a.connectAllDevices()

	return nil
}

// Request connect to every device
func (a *JoyconAPI) connectAllDevices() {
	var deviceList []dbus.ObjectPath
	a.mu.Lock()
	for path, info := range a.devicePaths {
		if info.IsJoyCon && !info.Connected {
			deviceList = append(deviceList, path)
		}
	}
	a.mu.Unlock()

	ch := make(chan *dbus.Call, len(deviceList))
	for _, path := range deviceList {
		a.busConn.Object(BlueZBusName, path).Go("org.bluez.Device1.ConnectProfile", 0, ch, HIDProfileUUIDStrU)
	}
	go func() {
		for _ = range deviceList {
			call := <-ch
			if call.Err != nil {
				fmt.Fprintf(os.Stderr,
					"\r[bluez] [INFO] failed to connect bluetooth device %s: %v\n",
					strings.TrimPrefix(string(call.Path), "/org/bluez/"),
					call.Err,
				)
			} else {
				a.checkDevice(call.Path)
			}
		}
	}()
	return
}

// Same as above, but only one device. Use 'go' when calling.
func (a *JoyconAPI) tryConnectDevice(path dbus.ObjectPath) {
	fmt.Println("[bluez] attempting connect to", path)
	call := a.busConn.Object(BlueZBusName, path).Call("org.bluez.Device1.ConnectProfile", 0, HIDProfileUUIDStrU)
	if call.Err != nil {
		fmt.Fprintf(os.Stderr,
			"\r[bluez] [INFO] failed to connect bluetooth device %s: %v\n",
			strings.TrimPrefix(string(call.Path), "/org/bluez/"),
			call.Err,
		)
	} else {
		// should get signal notified
		//a.checkDevice(call.Path)
		fmt.Println("[bluez] connect success to", path)
	}
}

func (a *JoyconAPI) tryPairDevice(path dbus.ObjectPath) {
	fmt.Println("[bluez] attempting pair with", path)
	call := a.busConn.Object(BlueZBusName, path).Call("org.bluez.Device1.Pair", 0)
	if call.Err != nil {
		fmt.Fprintf(os.Stderr,
			"\r[bluez] [INFO] failed to pair bluetooth device %s: %v\n",
			strings.TrimPrefix(string(call.Path), "/org/bluez/"),
			call.Err,
		)
	} else {
		//a.checkDevice(call.Path)
		fmt.Println("[bluez] pair success to", path)
	}
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
	//wasEnabled := a.discoveryEnabled
	if a.discoveryEnabled {
		a.discoveryEnabled = false
	}
	adapterList := a.adapterPaths
	a.mu.Unlock()

	fmt.Println("[bluez] stopping bt discovery")
	// Request discovery from every adapter
	ch := make(chan *dbus.Call, len(adapterList))
	for _, path := range adapterList {
		a.busConn.Object(BlueZBusName, path).Go("org.bluez.Adapter1.StopDiscovery", 0, ch)
	}
	for _ = range adapterList {
		call := <-ch
		if call.Err != nil {
			fmt.Fprintf(os.Stderr,
				"\r[bluez] failed to start bluetooth discovery for %s: %v\n",
				strings.TrimPrefix(string(call.Path), "/org/bluez/"),
				call.Err,
			)
		}
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
	// TODO
	// listBluetoothDevices()
	// for dev := devices
	// if isJoyCon(dev)
	// getAdapter(dev.adapter).RemoveDevice(path)
	fmt.Println("[bluez] [ERR] DeletePairingInfo: NOT IMPLEMENTED")
	return nil
}

// marks the device as Trusted
func (a *JoyconAPI) SavePairingInfo(mac [6]byte) {
	macStr := fmt.Sprintf("%02X_%02X_%02X_%02X_%02X_%02X",
		mac[0], mac[1], mac[2], mac[3], mac[4], mac[5],
	)

	go a.savePairingInfo(macStr)
}

func (a *JoyconAPI) savePairingInfo(macStr string) {
	// set Trusted=true
	var paths []dbus.ObjectPath
	a.mu.Lock()
	for k := range a.devicePaths {
		if strings.HasSuffix(string(k), macStr) {
			paths = append(paths, k)
		}
	}
	a.mu.Unlock()
	if len(paths) == 0 {
		fmt.Fprintf(os.Stderr, "\r[bluez] failed to save bluetooth pairing info for %s: device not found\n", macStr)
		return
	}
	for _, path := range paths {
		obj := a.busConn.Object(BlueZBusName, path)
		c := obj.Call("org.freedesktop.DBus.Properties.Set", 0, "org.bluez.Device1", "Trusted", true)
		if c.Err != nil {
			fmt.Fprintf(os.Stderr, "\r[bluez] failed to save bluetooth pairing info for %s: %v\n", path, c.Err)
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
	sigCall := busObj.Call("org.freedesktop.DBus.AddMatch", 0,
		"type='signal',sender='org.bluez',interface='org.freedesktop.DBus.ObjectManager',path='/'",
	)
	if sigCall.Err != nil {
		return errors.Wrap(sigCall.Err, "subscribe to updates")
	}
	// TODO - do this per adapter?
	sigCall = busObj.Call("org.freedesktop.DBus.AddMatch", 0,
		"type='signal',sender='org.bluez',interface='org.freedesktop.DBus.Properties',member='PropertiesChanged',path_namespace='/org/bluez'",
	)
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
		go a.checkDBusNewObject(path, v)
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

func (a *JoyconAPI) handleChangeSignals() {
	for busSig := range a.busSignalCh {
		if busSig.Name == "org.freedesktop.DBus.ObjectManager.InterfacesAdded" {
			var path dbus.ObjectPath
			var data dbusObjectNotify
			err := dbus.Store(busSig.Body, &path, &data)
			if err != nil {
				fmt.Fprintf(os.Stderr, "\r[bluez] failed to process InterfacesAdded message: %v\n%v\n", err, busSig)
				continue
			}
			fmt.Println("[bluez] [DEBUG] InterfacesAdded", path, data)
			go a.checkDBusNewObject(path, data)
		} else if busSig.Name == "org.freedesktop.DBus.ObjectManager.InterfacesRemoved" {
			var path dbus.ObjectPath
			var ifaces []string
			err := dbus.Store(busSig.Body, &path, &ifaces)
			if err != nil {
				fmt.Fprintf(os.Stderr, "\r[bluez] failed to process InterfacesRemoved message: %v\n%v\n", err, busSig)
				continue
			}
			fmt.Println("[bluez] [DEBUG] InterfacesRemoved", path, ifaces)
			go a.processRemoval(path, ifaces)
		} else if busSig.Name == "org.freedesktop.DBus.Properties.PropertiesChanged" {
			var path dbus.ObjectPath
			var iface string
			var changed map[string]dbus.Variant
			var invalidated []string
			path = busSig.Path
			err := dbus.Store(busSig.Body, &iface, &changed, &invalidated)
			if err != nil {
				fmt.Fprintf(os.Stderr, "\r[bluez] failed to process PropertiesChanged message: %v\n%v\n", err, busSig)
				continue
			}
			fmt.Println("[bluez] [DEBUG] PropertiesChanged", path, iface, changed, invalidated)
			// TODO
			go a.checkProperties(path, iface, changed, invalidated)
		} else {
			fmt.Println("[bluez]", "unhandled dbus signal", busSig)
		}
	}
}

func (a *JoyconAPI) checkDBusNewObject(path dbus.ObjectPath, data dbusObjectNotify) {
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
		a.checkDevice(path)
	}
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
			fmt.Println("[bluez] removed adapter", path)
		}
		if iface == Device1Interface {
			a.mu.Lock()
			devInfo := a.devicePaths[path]
			a.mu.Unlock()

			if devInfo.IsJoyCon {
				a.emitNotify(path, false, false)
				fmt.Println("[bluez] removed joy-con", path)
			} else {
				fmt.Println("[bluez] ignored device removal", path)
			}
		}
	}
}

func (a *JoyconAPI) checkProperties(path dbus.ObjectPath, iface string, changed map[string]dbus.Variant, invalidated []string) {
	if iface == Device1Interface {
		// this is wasteful but whatever
		a.checkDevice(path)
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
	a.mu.Lock()
	prevInfo := a.devicePaths[path]
	a.mu.Unlock()

	info, err := a.getDeviceInfo(path)
	if err != nil {
		fmt.Println("[bluez]", "device", path, "error", err)
	} else {
		a.mu.Lock()
		a.devicePaths[path] = info
		isDiscovering := a.discoveryEnabled
		a.mu.Unlock()

		if !info.IsJoyCon {
			return
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		fmt.Println("[bluez]", "device", path, "info")
		enc.Encode(info)

		if prevInfo.Connected != info.Connected {
			fmt.Println("[bluez] [info] notifying ui of connection state")
			// definition of "new controller" -- i.e. needs L+R press -- is !Trusted
			a.emitNotify(path, info.Connected, true)
		}
		if isDiscovering && prevInfo.IsJoyCon != info.IsJoyCon {
			// This is a new device
			fmt.Println("[bluez] checkDevice: attempt pair?", path)
			a.tryPairDevice(path)
		}
		if prevInfo.Paired != info.Paired && info.Paired {
			// Just completed pairing, autoconnect
			fmt.Println("[bluez] checkDevice: just paired, doing connect", path)
			a.tryConnectDevice(path)
		}
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

	// do we actually need destructured MAC here
	/*
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
	*/
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
