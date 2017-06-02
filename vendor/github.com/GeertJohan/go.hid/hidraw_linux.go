package hid

/**
 * This code is licensed under a Simplified BSD License. For more information read the LICENSE file that came with this package.
 *
 * File: hidraw_linux.go
 * Copyright (c) Kane York 2017
 */

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"syscall"
	"github.com/jochenvg/go-udev"
	"github.com/pkg/errors"
	"golang.org/x/sys/unix"
	"unsafe"
)

/*
#include <stdlib.h>
#include <linux/input.h>
#include <linux/hidraw.h>

int get_ioctl_set_feature(int length) {
	return HIDIOCSFEATURE(length);
}
int get_ioctl_get_feature(int length) {
	return HIDIOCGFEATURE(length);
}

*/
import "C"

type Device struct {
	handle io.ReadWriteCloser
	fd     int

	serial      string
	productName string
	// No need to work around a kernel bug fixed in 2.6.34
	// uses_numbered_reports int

	// By not calling into cgo for our writes, callers can use goroutines for non-blocking reads/writes
	// blocking bool
}

func (di *DeviceInfo) Device() (*Device, error) {
	d, err := OpenPath(di.Path)
	if err != nil {
		return nil, err
	}
	d.serial = di.SerialNumber
	d.productName = di.Product
	return d, nil
}

func parse_uevent_info(uevent string, bus_type *int, out *DeviceInfo) error {
	var found_id, found_serial, found_name bool

	for _, line := range strings.Split(uevent, "\n") {
		split := strings.SplitN(line, "=", 2)
		if len(split) != 2 {
			continue
		}
		key := split[0]
		value := split[1]
		switch key {
		case "HID_ID":
			/**
			 *        type vendor   product
			 * HID_ID=0003:000005AC:00008242
			 **/
			n, _ := fmt.Sscanf(value, "%x:%hx:%hx", &bus_type, &out.VendorId, &out.ProductId)
			if n == 3 {
				found_id = true
			}
		case "HID_NAME":
			out.Product = value
			found_name = true
		case "HID_UNIQ":
			out.SerialNumber = value
			found_serial = true
		}
	}

	if !found_id || !found_name || !found_serial {
		return errors.Errorf("Udev info parse error: [[%s]]: values missing", uevent)
	}
	return nil
}

// Retrieve a list of DeviceInfo objects that match the given vendorId and productId.
// To retrieve a list of all HID devices': use 0x0 as vendorId and productId.
func Enumerate(vendorId uint16, productId uint16) (DeviceInfoList, error) {
	u := udev.Udev{}
	udevEnum := u.NewEnumerate()
	udevEnum.AddMatchSubsystem("hidraw")
	it, err := udevEnum.DeviceIterator()
	if err != nil {
		return nil, err
	}

	var list DeviceInfoList

	it.Each(func(v_ interface{}) {
		dev := v_.(*udev.Device)
		fmt.Println(dev)

		hid_dev := dev.ParentWithSubsystemDevtype("hid", "")
		if hid_dev == nil {
			// Unable to find parent HID device; continue.
			return
		}

		di := &DeviceInfo{
			Path: dev.Devnode(),
		}

		var bus_type int
		// di.VendorId
		// di.ProductId
		// di.Product
		// di.SerialNumber
		err = parse_uevent_info(hid_dev.SysattrValue("uevent"), &bus_type, di)
		if err != nil {
			// Skip.
			return
		}

		if bus_type != C.BUS_USB && bus_type != C.BUS_BLUETOOTH {
			// We only know how to handle USB and Bluetooth.
			return
		}

		if ((vendorId != 0) && (vendorId != di.VendorId)) ||
			((productId != 0) && (productId != di.ProductId)) {
			// Vendor / Product ID mismatch
			return
		}

		di.ReleaseNumber = 0
		di.InterfaceNumber = -1

		if bus_type == C.BUS_USB {
			/* The device pointed to by hid_dev contains information about
			   the hidraw device. In order to get information about the
			   USB device, get the parent device with the
			   subsystem/devtype pair of "usb"/"usb_device". This will
			   be several levels up the tree, but the function will find
			   it. */
			usbDev := hid_dev.ParentWithSubsystemDevtype("usb", "usb_device")
			if usbDev == nil {
				// Fake USB device, skip.
				return
			}

			di.Manufacturer = usbDev.SysattrValue("manufacturer")
			// TODO check consequences of this line
			di.Product = usbDev.SysattrValue("product")

			releaseNumber, err := strconv.ParseUint(usbDev.SysattrValue("bcdDevice"), 16, 16)
			if err != nil {
				di.ReleaseNumber = 0
			} else {
				di.ReleaseNumber = uint16(releaseNumber)
			}

			interfaceDev := hid_dev.ParentWithSubsystemDevtype("usb", "usb_interface")
			if interfaceDev != nil {
				interfaceNumber, err := strconv.ParseInt(usbDev.SysattrValue("bInterfaceNumber"), 16, 32)
				if err != nil {
					di.InterfaceNumber = -1
				} else {
					di.InterfaceNumber = int(interfaceNumber)
				}
			}
		} else /* bus_type == BUS_BLUETOOTH */ {
			// manufacturer string, etc not available without querying device
			// only available strings:
			// uevent, report_descriptor (bytes), modalias, country
		}

		list = append(list, di)
	})
	it.Close()

	return list, nil
}

// Open hid by path.
// Returns a *Device and an error
func OpenPath(path string) (*Device, error) {
	file, err := unix.Open(path, unix.O_RDWR, 0)
	if err != nil {
		return nil, err
	}

	return &Device{
		handle: os.NewFile(uintptr(file), path),
		fd:     file,
	}, nil
}

func (dev *Device) Write(p []byte) (n int, err error) {
	return dev.handle.Write(p)
}

func (dev *Device) Read(p []byte) (n int, err error) {
	return dev.handle.Read(p)
}

func (dev *Device) SendFeatureReport(data []byte) (int, error) {
	ptr := C.malloc(C.size_t(len(data)))
	defer C.free(ptr)

	cBuf := (*[1 << 30]byte)(ptr)
	copy(cBuf[:], data)

	ret, _, err := unix.Syscall(syscall.SYS_IOCTL,
		uintptr(dev.fd),
		uintptr(C.get_ioctl_set_feature(C.int(len(data)))),
		uintptr(ptr))

	if int(ret) == -1 {
		return -1, err
	}
	return int(ret), nil
}

func (dev *Device) GetFeatureReport(reportId byte, reportDataSize int) ([]byte, error) {
	reportSize := reportDataSize + 1
	buf := make([]byte, reportSize)
	buf[0] = reportId
	bufPtr := (*C.uchar)(&buf[0])

	ret, _, err := unix.Syscall(syscall.SYS_IOCTL,
		uintptr(dev.fd),
		uintptr(C.get_ioctl_get_feature(C.int(reportSize))),
		uintptr(unsafe.Pointer(bufPtr)))

	if int(ret) == -1 {
		return nil, err
	}
	return buf, nil
}

func (dev *Device) Close() error {
	dev.handle.Close()
	dev.handle = nil
	dev.fd = -1
	return nil
}

func (dev *Device) SerialNumberString() (string, error) {
	return dev.serial, nil
}

func (dev *Device) GetIndexedString(index int) (string, error) {
	return "", errors.New("Not supported on Linux")
}

// In non-blocking mode calls to hid_read() will return immediately with a value of 0 if there is no data to be read.
// In blocking mode, hid_read() will wait (block) until there is data to read before returning.
//
// On Linux, SetReadWriteNonBlocking is ignored because the Read/Write calls are dispatched by the Go runtime,
// so handle concurrency correctly.
func (dev *Device) SetReadWriteNonBlocking(nonblocking bool) error {
	return nil
}
