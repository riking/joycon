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
	"strings"

	"github.com/jochenvg/go-udev"
	"github.com/pkg/errors"
)

/*
#include <linux/input.h>
 */
import "C"

type Device struct {
	handle                io.ReadWriteCloser
	uses_numbered_reports int
	// blocking bool
}

// In non-blocking mode calls to hid_read() will return immediately with a value of 0 if there is no data to be read.
// In blocking mode, hid_read() will wait (block) until there is data to read before returning.
//
// On Linux, SetReadWriteNonBlocking is ignored because the Read/Write calls are dispatched by the Go runtime,
// so handle concurrency correctly.
func (dev *Device) SetReadWriteNonBlocking(nonblocking bool) error {
	return nil
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

	it.Each(func(v_ interface{}) {
		dev := v_.(*udev.Device)

		hid_dev := dev.ParentWithSubsystemDevtype("hid", "")
		if hid_dev == nil {
			// Unable to find parent HID device; continue.
			return
		}

		di := &DeviceInfo{
			Path: dev.Devnode(),
		}

		var bus_type int
		err = parse_uevent_info(hid_dev.SysattrValue("uevent"), &bus_type, di)

		if bus_type != C.BUS_USB && bus_type != C.BUS_BLUETOOTH {
			// We only know how to handle USB and Bluetooth.
			return
		}
		//VendorId:        uint16(next.vendor_id),
		//ProductId:       uint16(next.product_id),
		//ReleaseNumber:   uint16(next.release_number),
		//UsagePage:       uint16(next.usage_page),
		//Usage:           uint16(next.usage),
		//InterfaceNumber: int(next.interface_number),
	})
	it.Close()
}
