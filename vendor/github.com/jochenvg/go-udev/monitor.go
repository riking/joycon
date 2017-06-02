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
	"syscall"

	"golang.org/x/sys/unix"
)

// Monitor is an opaque object handling an event source
type Monitor struct {
	ptr *C.struct_udev_monitor
	u   *Udev
}

const (
	maxEpollEvents = 32
	epollTimeout   = 1000
)

// Lock the udev context
func (m *Monitor) lock() {
	m.u.m.Lock()
}

// Unlock the udev context
func (m *Monitor) unlock() {
	m.u.m.Unlock()
}

// Unref the monitor
func monitorUnref(m *Monitor) {
	C.udev_monitor_unref(m.ptr)
}

// SetReceiveBufferSize sets the size of the kernel socket buffer.
// This call needs the appropriate privileges to succeed.
func (m *Monitor) SetReceiveBufferSize(size int) (err error) {
	m.lock()
	defer m.unlock()
	if C.udev_monitor_set_receive_buffer_size(m.ptr, (C.int)(size)) != 0 {
		err = errors.New("udev: udev_monitor_set_receive_buffer_size failed")
	}
	return
}

// FilterAddMatchSubsystem adds a filter matching the device against a subsystem.
// This filter is efficiently executed inside the kernel, and libudev subscribers will usually not be woken up for devices which do not match.
// The filter must be installed before the monitor is switched to listening mode with the DeviceChan function.
func (m *Monitor) FilterAddMatchSubsystem(subsystem string) (err error) {
	m.lock()
	defer m.unlock()
	s := C.CString(subsystem)
	defer freeCharPtr(s)
	if C.udev_monitor_filter_add_match_subsystem_devtype(m.ptr, s, nil) != 0 {
		err = errors.New("udev: udev_monitor_filter_add_match_subsystem_devtype failed")
	}
	return
}

// FilterAddMatchSubsystemDevtype adds a filter matching the device against a subsystem and device type.
// This filter is efficiently executed inside the kernel, and libudev subscribers will usually not be woken up for devices which do not match.
// The filter must be installed before the monitor is switched to listening mode with the DeviceChan function.
func (m *Monitor) FilterAddMatchSubsystemDevtype(subsystem, devtype string) (err error) {
	m.lock()
	defer m.unlock()
	s, d := C.CString(subsystem), C.CString(devtype)
	defer freeCharPtr(s)
	defer freeCharPtr(d)
	if C.udev_monitor_filter_add_match_subsystem_devtype(m.ptr, s, d) != 0 {
		err = errors.New("udev: udev_monitor_filter_add_match_subsystem_devtype failed")
	}
	return
}

// FilterAddMatchTag adds a filter matching the device against a tag.
// This filter is efficiently executed inside the kernel, and libudev subscribers will usually not be woken up for devices which do not match.
// The filter must be installed before the monitor is switched to listening mode.
func (m *Monitor) FilterAddMatchTag(tag string) (err error) {
	m.lock()
	defer m.unlock()
	t := C.CString(tag)
	defer freeCharPtr(t)
	if C.udev_monitor_filter_add_match_tag(m.ptr, t) != 0 {
		err = errors.New("udev: udev_monitor_filter_add_match_tag failed")
	}
	return
}

// FilterUpdate updates the installed socket filter.
// This is only needed, if the filter was removed or changed.
func (m *Monitor) FilterUpdate() (err error) {
	m.lock()
	defer m.unlock()
	if C.udev_monitor_filter_update(m.ptr) != 0 {
		err = errors.New("udev: udev_monitor_filter_update failed")
	}
	return
}

// FilterRemove removes all filter from the Monitor.
func (m *Monitor) FilterRemove() (err error) {
	m.lock()
	defer m.unlock()
	if C.udev_monitor_filter_remove(m.ptr) != 0 {
		err = errors.New("udev: udev_monitor_filter_remove failed")
	}
	return
}

// receiveDevice is a helper function receiving a device while the Mutex is locked
func (m *Monitor) receiveDevice() (d *Device) {
	m.lock()
	defer m.unlock()
	return m.u.newDevice(C.udev_monitor_receive_device(m.ptr))
}

// DeviceChan binds the udev_monitor socket to the event source and spawns a
// goroutine. The goroutine efficiently waits on the monitor socket using epoll.
// Data is received from the udev monitor socket and a new Device is created
// with the data received. Pointers to the device are sent on the returned channel.
// The function takes a done signalling channel as a parameter, which when
// triggered will stop the goroutine and close the device channel.
// Only socket connections with uid=0 are accepted.
func (m *Monitor) DeviceChan(done <-chan struct{}) (<-chan *Device, error) {

	var event unix.EpollEvent
	var events [maxEpollEvents]unix.EpollEvent

	// Lock the context
	m.lock()
	defer m.unlock()

	// Enable receiving
	if C.udev_monitor_enable_receiving(m.ptr) != 0 {
		return nil, errors.New("udev: udev_monitor_enable_receiving failed")
	}

	// Set the fd to non-blocking
	fd := C.udev_monitor_get_fd(m.ptr)
	if e := unix.SetNonblock(int(fd), true); e != nil {
		return nil, errors.New("udev: unix.SetNonblock failed")
	}

	// Create an epoll fd
	epfd, e := unix.EpollCreate1(0)
	if e != nil {
		return nil, errors.New("udev: unix.EpollCreate1 failed")
	}

	// Add the fd to the epoll fd
	event.Events = unix.EPOLLIN | unix.EPOLLET
	event.Fd = int32(fd)
	if e = unix.EpollCtl(epfd, unix.EPOLL_CTL_ADD, int(fd), &event); e != nil {
		return nil, errors.New("udev: unix.EpollCtl failed")
	}

	// Create the channel
	ch := make(chan *Device)

	// Create goroutine to epoll the fd
	go func(done <-chan struct{}, fd int32) {
		// Close the epoll fd when goroutine exits
		defer unix.Close(epfd)
		// Close the channel when goroutine exits
		defer close(ch)
		// Loop forever
		for {
			// Poll the file descriptor
			nevents, e := unix.EpollWait(epfd, events[:], epollTimeout)
			if e != nil {
				if errno, ok := e.(syscall.Errno); ok {
					if errno == syscall.EINTR {
						continue
					}
				}
				return
			}
			// Process events
			for ev := 0; ev < nevents; ev++ {
				if events[ev].Fd == fd {
					if (events[ev].Events & unix.EPOLLIN) != 0 {
						for d := m.receiveDevice(); d != nil; d = m.receiveDevice() {
							ch <- d
						}
					}
				}
			}
			// Check for done signal
			select {
			case <-done:
				return
			default:
			}
		}
	}(done, int32(fd))

	return ch, nil
}
