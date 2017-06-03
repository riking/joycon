package hid

import (
	"errors"
	"sync"
	"fmt"
)

var errNotImplemented = errors.New("not implemented yet")

//++ FIXME: How to do this on binary end? C.hid_exit()
// wrap as hid.Exit() ???? that sounds bad..

// /** hidapi info structure */
// struct hid_device_info {
//  /** Platform-specific device path */
//  char *path;
//  /** Device Vendor ID */
//  unsigned short vendor_id;
//  /** Device Product ID */
//  unsigned short product_id;
//  /** Serial Number */
//  wchar_t *serial_number;
//  /** Device Release Number in binary-coded decimal,
//      also known as Device Version Number */
//  unsigned short release_number;
//  /** Manufacturer String */
//  wchar_t *manufacturer_string;
//  /** Product string */
//  wchar_t *product_string;
//  /** Usage Page for this Device/Interface
//      (Windows/Mac only). */
//  unsigned short usage_page;
//  /** Usage for this Device/Interface
//      (Windows/Mac only).*/
//  unsigned short usage;
//  /** The USB interface which this logical device
//      represents. Valid on both Linux implementations
//      in all cases, and valid on the Windows implementation
//      only if the device contains more than one interface. */
//  int interface_number;
//  /** Pointer to the next device */
//  struct hid_device_info *next;
// };

// DeviceInfo provides all information about an HID device.
type DeviceInfo struct {
	Path            string
	VendorId        uint16
	ProductId       uint16
	SerialNumber    string
	ReleaseNumber   uint16
	Manufacturer    string
	Product         string
	UsagePage       uint16 // Only being used with windows/mac, which are not supported by go.hid yet.
	Usage           uint16 // Only being used with windows/mac, which are not supported by go.hid yet.
	InterfaceNumber int
}

// List of DeviceInfo objects
type DeviceInfoList []*DeviceInfo

var initOnce sync.Once


type wrapError struct {
	w   error
	ctx string
}

func (w wrapError) Cause() error {
	return w.w
}

func (w wrapError) Error() string {
	return fmt.Sprintf("%s: %v", w.ctx, w.w)
}
