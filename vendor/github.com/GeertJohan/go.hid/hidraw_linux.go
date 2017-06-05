package hid

/**
 * This code is licensed under a Simplified BSD License. For more information read the LICENSE file that came with this package.
 *
 * File: hidraw_linux.go
 * Copyright (c) Kane York 2017
 */

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
	"unsafe"

	"github.com/jochenvg/go-udev"
	"github.com/pkg/errors"
	"golang.org/x/sys/unix"
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
	epoll  int
	fd     int
	closed bool
	grab   bool

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
			n, _ := fmt.Sscanf(value, "%x:%x:%x", bus_type, &out.VendorId, &out.ProductId)
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
	fd, err := unix.Open(path, unix.O_RDWR|unix.O_NONBLOCK, 0)
	if err != nil {
		return nil, err
	}

	pollFd, err := unix.EpollCreate1(0)
	if err != nil {
		unix.Close(fd)
		return nil, errors.Wrap(err, "epoll_create")
	}
	err = unix.EpollCtl(pollFd, unix.EPOLL_CTL_ADD, fd, &unix.EpollEvent{
		Events: unix.EPOLLIN | unix.EPOLLOUT,
	})
	if err != nil {
		unix.Close(pollFd)
		unix.Close(fd)
		return nil, errors.Wrap(err, "epoll_ctl")
	}

	return &Device{
		epoll: pollFd,
		fd:    fd,
	}, nil
}

func (dev *Device) Write(p []byte) (n int, err error) {
	if dev == nil || dev.closed {
		return -1, os.ErrClosed
	}

	for attempts := 4; attempts > 0; attempts-- {
		n, err = unix.Write(dev.fd, p)
		if err == unix.EAGAIN {
			_, pErr := unix.EpollWait(dev.epoll, []unix.EpollEvent{{
				Events: unix.EPOLLOUT,
				Fd:     int32(dev.fd),
			}}, 1)
			if pErr != nil {
				fmt.Println("poll error:", pErr)
			}
			continue
		} else if err != nil {
			return n, &os.PathError{Err: err, Op: "write", Path: dev.serial}
		}
		return n, err
	}
	return -1, err
}

func (dev *Device) Read(p []byte) (n int, err error) {
	if dev == nil || dev.closed {
		return -1, os.ErrClosed
	}

	for attempts := 4; attempts > 0; attempts-- {
		n, err = unix.Read(dev.fd, p)
		if err == unix.EAGAIN {
			_, pErr := unix.EpollWait(dev.epoll, []unix.EpollEvent{{
				Events: unix.EPOLLIN,
				Fd:     int32(dev.fd),
			}}, 16)
			if pErr != nil {
				fmt.Println("poll error:", pErr)
			}
			continue
		} else if err != nil {
			return n, &os.PathError{Err: err, Op: "read", Path: dev.serial}
		}
		return n, err
	}
	return 0, nil
}

func (dev *Device) SendFeatureReport(data []byte) (int, error) {
	if dev == nil || dev.closed {
		return -1, os.ErrClosed
	}

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
	if dev == nil || dev.closed {
		return nil, os.ErrClosed
	}

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

// In non-blocking mode calls to hid_read() will return immediately with a value of 0 if there is no data to be read.
// In blocking mode, hid_read() will wait (block) until there is data to read before returning.
func (dev *Device) SetReadWriteNonBlocking(nonblocking bool) error {
	state := uintptr(0)
	if nonblocking {
		state = 1
	}
	status, _, err := unix.Syscall6(syscall.SYS_IOCTL,
		uintptr(dev.fd),
		uintptr(unix.F_SETFL),
		uintptr(unix.O_NONBLOCK),
		state,
		0, 0)

	if status != 0 {
		return err
	}
	return nil
}

func (dev *Device) AttemptGrab(grab bool) error {
	var param C.int = 0
	if grab {
		param = 1
	}
	return nil
	fmt.Println("grab", grab)
	status, _, err := unix.Syscall(syscall.SYS_IOCTL,
		uintptr(dev.fd),
		uintptr(C.EVIOCGRAB),
		uintptr(unsafe.Pointer(&param)))
	if status != 0 {
		fmt.Println("grab error", err)
		return err
	}
	dev.grab = grab
	return nil
}

func (dev *Device) Close() error {
	if dev == nil || dev.closed {
		return os.ErrClosed
	}

	fmt.Println("closing uinput", dev.fd, dev.epoll, dev.grab)
	dev.AttemptGrab(false)

	unix.Close(dev.fd)
	unix.Close(dev.epoll)
	dev.closed = true
	return nil
}

func (dev *Device) SerialNumberString() (string, error) {
	return dev.serial, nil
}

func (dev *Device) GetIndexedString(index int) (string, error) {
	return "", errors.New("Not supported on Linux")
}
