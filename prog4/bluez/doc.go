// bluez package handles interaction with dbus for Bluetooth discovery on
// Linux.
//
// plan: open /org/bluez. call Introspect to get the full list of objects.
// (but also subscribe to change notifications)
// we don't care about GattService, just Adapter and Device
// how to identify unpaired devices? MAC? Name? (obviously .Paired==false)
//
// (!) Set all controllers to Trusted for easy autoreconnect.  The controller
// will only "trust" the last device to assign it a player number
//
// Need a RemoveAllSyncRecords() function - if bluez is holding on to old
// pairing record, device needs to be deleted
//
// errors from Connect():
// already connected: "device busy"
// generic (not in range..): "i/o error (36)"
//
// when a Device1 is found or changes:
//   1. if adapter is blacklisted, skip
//   1. if .Connected and .Trusted, everything is fine. emit an input device recheck with MAC
//   2. if not .Paired:
//     send an async Pair() method
//     when that returns, if success or "already exists":
//     send ConnectProfile(HID) method (? DEBUG THIS)
//     wait for .Connected changes (algorithm restarts)
//   3. if .Connected up but not .Trusted:
//     emit an unpaired input device recheck with MAC
//     -> once L+R is pressed, we set .Trusted to true
//
package bluez

import "sync"
import "github.com/riking/joycon/prog4/jcpc"

var autoconnectDeviceNames = []string{
	"Pro Controller",
	"Joy-Con (L)",
	"Joy-Con (R)",
}

// The Bluetooth profile of interest to us.
const (
	HIDProfileShort   = 0x1124
	HIDProfileUUIDStr = "00001124-0000-1000-8000-00805F9B34FB"
)
