// +build linux,cgo

package udev

/*
  #cgo LDFLAGS: -ludev
  #include <libudev.h>
  #include <linux/types.h>
  #include <stdlib.h>
	#include <linux/kdev_t.h>
*/
import "C"

import (
	"errors"

	"github.com/jkeiser/iter"
)

// Enumerate is an opaque struct wrapping a udev enumerate object.
type Enumerate struct {
	ptr *C.struct_udev_enumerate
	u   *Udev
}

// Lock the udev context
func (e *Enumerate) lock() {
	e.u.m.Lock()
}

// Unlock the udev context
func (e *Enumerate) unlock() {
	e.u.m.Unlock()
}

// Unref the Enumerate object
func enumerateUnref(e *Enumerate) {
	C.udev_enumerate_unref(e.ptr)
}

// AddMatchSubsystem adds a filter for a subsystem of the device to include in the list.
func (e *Enumerate) AddMatchSubsystem(subsystem string) (err error) {
	e.lock()
	defer e.unlock()
	s := C.CString(subsystem)
	defer freeCharPtr(s)
	if C.udev_enumerate_add_match_subsystem(e.ptr, s) != 0 {
		err = errors.New("udev: udev_enumerate_add_match_subsystem failed")
	}
	return
}

// AddNomatchSubsystem adds a filter for a subsystem of the device to exclude from the list.
func (e *Enumerate) AddNomatchSubsystem(subsystem string) (err error) {
	e.lock()
	defer e.unlock()
	s := C.CString(subsystem)
	defer freeCharPtr(s)
	if C.udev_enumerate_add_nomatch_subsystem(e.ptr, s) != 0 {
		err = errors.New("udev: udev_enumerate_add_nomatch_subsystem failed")
	}
	return
}

// AddMatchSysattr adds a filter for a sys attribute at the device to include in the list.
func (e *Enumerate) AddMatchSysattr(sysattr, value string) (err error) {
	e.lock()
	defer e.unlock()
	s, v := C.CString(sysattr), C.CString(value)
	defer freeCharPtr(s)
	defer freeCharPtr(v)
	if C.udev_enumerate_add_match_sysattr(e.ptr, s, v) != 0 {
		err = errors.New("udev: udev_enumerate_add_match_sysattr failed")
	}
	return
}

// AddNomatchSysattr adds a filter for a sys attribute at the device to exclude from the list.
func (e *Enumerate) AddNomatchSysattr(sysattr, value string) (err error) {
	e.lock()
	defer e.unlock()
	s, v := C.CString(sysattr), C.CString(value)
	defer freeCharPtr(s)
	defer freeCharPtr(v)
	if C.udev_enumerate_add_nomatch_sysattr(e.ptr, s, v) != 0 {
		err = errors.New("udev: udev_enumerate_add_nomatch_sysattr failed")
	}
	return
}

// AddMatchProperty adds a filter for a property of the device to include in the list.
func (e *Enumerate) AddMatchProperty(property, value string) (err error) {
	e.lock()
	defer e.unlock()
	p, v := C.CString(property), C.CString(value)
	defer freeCharPtr(p)
	defer freeCharPtr(v)
	if C.udev_enumerate_add_match_property(e.ptr, p, v) != 0 {
		err = errors.New("udev: udev_enumerate_add_match_property failed")
	}
	return
}

// AddMatchSysname adds a filter for the name of the device to include in the list.
func (e *Enumerate) AddMatchSysname(sysname string) (err error) {
	e.lock()
	defer e.unlock()
	s := C.CString(sysname)
	defer freeCharPtr(s)
	if C.udev_enumerate_add_match_sysname(e.ptr, s) != 0 {
		err = errors.New("udev: udev_enumerate_add_match_sysname failed")
	}
	return
}

// AddMatchTag adds a filter for a tag of the device to include in the list.
func (e *Enumerate) AddMatchTag(tag string) (err error) {
	e.lock()
	defer e.unlock()
	t := C.CString(tag)
	defer freeCharPtr(t)
	if C.udev_enumerate_add_match_tag(e.ptr, t) != 0 {
		err = errors.New("udev: udev_enumerate_add_match_tag failed")
	}
	return
}

// AddMatchParent adds a filter for a parent Device to include in the list.
func (e *Enumerate) AddMatchParent(parent *Device) (err error) {
	e.lock()
	defer e.unlock()
	if C.udev_enumerate_add_match_parent(e.ptr, parent.ptr) != 0 {
		err = errors.New("udev: udev_enumerate_add_match_parent failed")
	}
	return
}

// AddMatchIsInitialized adds a filter matching only devices which udev has set up already.
// This makes sure, that the device node permissions and context are properly set and that network devices are fully renamed.
// Usually, devices which are found in the kernel but not already handled by udev, have still pending events.
// Services should subscribe to monitor events and wait for these devices to become ready, instead of using uninitialized devices.
// For now, this will not affect devices which do not have a device node and are not network interfaces.
func (e *Enumerate) AddMatchIsInitialized() (err error) {
	e.lock()
	defer e.unlock()
	if C.udev_enumerate_add_match_is_initialized(e.ptr) != 0 {
		err = errors.New("udev: udev_enumerate_add_match_is_initialized failed")
	}
	return
}

// AddSyspath adds a device to the list of enumerated devices, to retrieve it back sorted in dependency order.
func (e *Enumerate) AddSyspath(syspath string) (err error) {
	e.lock()
	defer e.unlock()
	s := C.CString(syspath)
	defer freeCharPtr(s)
	if C.udev_enumerate_add_syspath(e.ptr, s) != 0 {
		err = errors.New("udev: udev_enumerate_add_syspath failed")
	}
	return
}

// DeviceSyspaths retrieves a list of device syspaths matching the filter, sorted in dependency order.
func (e *Enumerate) DeviceSyspaths() (s []string, err error) {
	e.lock()
	defer e.unlock()
	if C.udev_enumerate_scan_devices(e.ptr) < 0 {
		err = errors.New("udev: udev_enumerate_scan_devices failed")
	} else {
		s = make([]string, 0)
		for l := C.udev_enumerate_get_list_entry(e.ptr); l != nil; l = C.udev_list_entry_get_next(l) {
			s = append(s, C.GoString(C.udev_list_entry_get_name(l)))
		}
	}
	return
}

// DeviceSyspathIterator returns an Iterator over the device syspaths matching the filter, sorted in dependency order.
// The Iterator is using the github.com/jkeiser/iter package.
// Values are returned as an interface{} and should be cast to string.
func (e *Enumerate) DeviceSyspathIterator() (it iter.Iterator, err error) {
	e.lock()
	defer e.unlock()
	if C.udev_enumerate_scan_devices(e.ptr) < 0 {
		err = errors.New("udev: udev_enumerate_scan_devices failed")
	} else {
		l := C.udev_enumerate_get_list_entry(e.ptr)
		it = iter.Iterator{
			Next: func() (item interface{}, err error) {
				e.lock()
				defer e.unlock()
				if l != nil {
					item = C.GoString(C.udev_list_entry_get_name(l))
					l = C.udev_list_entry_get_next(l)
				} else {
					err = iter.FINISHED
				}
				return
			},
			Close: func() {
			},
		}
	}
	return
}

// SubsystemSyspaths retrieves a list of subsystem syspaths matching the filter, sorted in dependency order.
func (e *Enumerate) SubsystemSyspaths() (s []string, err error) {
	e.lock()
	defer e.unlock()
	if C.udev_enumerate_scan_subsystems(e.ptr) < 0 {
		err = errors.New("udev: udev_enumerate_scan_subsystems failed")
	} else {
		s = make([]string, 0)
		for l := C.udev_enumerate_get_list_entry(e.ptr); l != nil; l = C.udev_list_entry_get_next(l) {
			s = append(s, C.GoString(C.udev_list_entry_get_name(l)))
		}
	}
	return
}

// DeviceSubsystemIterator returns an Iterator over the subsystem syspaths matching the filter, sorted in dependency order.
// The Iterator is using the github.com/jkeiser/iter package.
// Values are returned as an interface{} and should be cast to string.
func (e *Enumerate) DeviceSubsystemIterator() (it iter.Iterator, err error) {
	e.lock()
	defer e.unlock()
	if C.udev_enumerate_scan_subsystems(e.ptr) < 0 {
		err = errors.New("udev: udev_enumerate_scan_devices failed")
	} else {
		l := C.udev_enumerate_get_list_entry(e.ptr)
		it = iter.Iterator{
			Next: func() (item interface{}, err error) {
				e.lock()
				defer e.unlock()
				if l != nil {
					item = C.GoString(C.udev_list_entry_get_name(l))
					l = C.udev_list_entry_get_next(l)
				} else {
					err = iter.FINISHED
				}
				return
			},
			Close: func() {
			},
		}
	}
	return
}

// Devices retrieves a list of Devices matching the filter, sorted in dependency order.
func (e *Enumerate) Devices() (m []*Device, err error) {
	e.lock()
	defer e.unlock()
	if C.udev_enumerate_scan_devices(e.ptr) < 0 {
		err = errors.New("udev: udev_enumerate_scan_devices failed")
	} else {
		m = make([]*Device, 0)
		for l := C.udev_enumerate_get_list_entry(e.ptr); l != nil; l = C.udev_list_entry_get_next(l) {
			s := C.udev_list_entry_get_name(l)
			m = append(m, e.u.newDevice(C.udev_device_new_from_syspath(e.u.ptr, s)))
		}
	}
	return
}

// DeviceIterator returns an Iterator over the Devices matching the filter, sorted in dependency order.
// The Iterator is using the github.com/jkeiser/iter package.
// Values are returned as an interface{} and should be cast to *Device.
func (e *Enumerate) DeviceIterator() (it iter.Iterator, err error) {
	e.lock()
	defer e.unlock()
	if C.udev_enumerate_scan_devices(e.ptr) < 0 {
		err = errors.New("udev: udev_enumerate_scan_devices failed")
	} else {
		l := C.udev_enumerate_get_list_entry(e.ptr)
		it = iter.Iterator{
			Next: func() (item interface{}, err error) {
				e.lock()
				defer e.unlock()
				if l != nil {
					s := C.udev_list_entry_get_name(l)
					item = e.u.newDevice(C.udev_device_new_from_syspath(e.u.ptr, s))
					l = C.udev_list_entry_get_next(l)
				} else {
					err = iter.FINISHED
				}
				return
			},
			Close: func() {
			},
		}
	}
	return
}
