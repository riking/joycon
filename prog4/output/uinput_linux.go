package output

import (
	"encoding/binary"
	"fmt"
	"sync"
	"unsafe"

	"github.com/pkg/errors"
	"github.com/riking/joycon/prog4/jcpc"
	"golang.org/x/sys/unix"
	"syscall"
)

/*
#include <linux/input.h>
//#include <linux/uinput.h>
#include "uinput_linux.h"
#include <stddef.h>

static const struct input_event sample_ev;
const size_t offset_of_type = offsetof(struct input_event, type);
const size_t offset_of_code = offsetof(struct input_event, code);
const size_t offset_of_value = offsetof(struct input_event, value);

int write_uinput_setup(struct uinput_user_dev *setup, int fd) {

}

*/
import "C"

// gyro resolution is 4096 points/g because it's value of 4096 at rest
// To send gyro events, we need multiple event nodes (!)

// ??
const ff_effects_max = 1

type uinput struct {
	fd      int
	gyro_fd int

	buttons internalKeyCodeMapping
	axes    []commonStickMap

	// Locked by BeginUpdate, unlocked by FlushUpdate
	mu      sync.Mutex
	pending []uinputEvent
}

func (o *uinput) ui_ioctl(code, val uintptr) error {
	status, _, err := unix.Syscall(unix.SYS_IOCTL,
		uintptr(o.fd),
		uintptr(code),
		uintptr(val))
	if status != 0 {
		return err
	}
	return nil
}

func (o *uinput) ui_ioctl_r(code, val uintptr) (uintptr, error) {
	status, _, err := unix.Syscall(unix.SYS_IOCTL,
		uintptr(o.fd),
		uintptr(code),
		uintptr(val))
	if err != syscall.Errno(0) {
		return -1, err
	}
	return status, nil
}

type uinputEvent struct {
	Type  uint16
	Code  uint16
	Value int32
}

type internalKeyCodeMapping struct {
	KeyCodes [3 * 8]uint16 // 3 bytes * 8 bits -> uinput key code
}

const uinputEventSize = C.sizeof_struct_input_event

func (u uinputEvent) EncodeTo(p []byte) int {
	binary.LittleEndian.PutUint16(p[C.offset_of_type:], u.Type)
	binary.LittleEndian.PutUint16(p[C.offset_of_code:], u.Code)
	binary.LittleEndian.PutUint32(p[C.offset_of_value:], uint32(u.Value))

	return C.sizeof_struct_input_event
}

func (o *uinput) setupNewKernel(m ControllerMapping, name string) error {
	var setup C.struct_uinput_setup
	setup.id.bustype = C.BUS_BLUETOOTH
	setup.id.vendor = jcpc.VENDOR_NINTENDO
	setup.id.product = jcpc.JOYCON_PRODUCT_FAKE
	setup.id.version = 1
	setup.ff_effects_max = ff_effects_max
	for i, v := range []byte(name) {
		setup.name[i] = v
	}
	err := o.ui_ioctl(C.UI_DEV_SETUP, uintptr(unsafe.Pointer(&setup)))
	if err != nil {
		return errors.Wrap(err, "ioctl uinput_device_setup")
	}

	var abs_setup struct {
		code    uint16
		_pad    uint16
		absinfo struct {
			value      int32
			min        int32
			max        int32
			fuzz       int32
			flat       int32
			resolution int32
		}
	}
	abs_setup.absinfo.value = 0
	abs_setup.absinfo.min = -128
	abs_setup.absinfo.max = 128
	abs_setup.absinfo.fuzz = 2
	abs_setup.absinfo.flat = 2
	o.axes = m.Axes
	for _, e := range o.axes {
		if e.Name == "" {
			continue
		}
		code, ok := linuxKeyMap[e.Name]
		if !ok {
			return errors.Errorf("Unrecognized axis name '%s'", e.Name)
		}
		abs_setup.code = code
		err = o.ui_ioctl(C.UI_ABS_SETUP, uintptr(unsafe.Pointer(&abs_setup)))
		if err != nil {
			return errors.Wrap(err, "ioctl uinput_abs_setup")
		}
	}
	return nil
}

func (o *uinput) setupOldKernel(m ControllerMapping, name string) error {
	var setup C.struct_uinput_user_dev
	setup.id.bustype = C.BUS_BLUETOOTH
	setup.id.vendor = jcpc.VENDOR_NINTENDO
	setup.id.product = jcpc.JOYCON_PRODUCT_FAKE
	setup.id.version = 1
	for i, v := range []byte(name) {
		setup.name[i] = v
	}
	setup.ff_effects_max = ff_effects_max

	o.axes = m.Axes
	for _, e := range o.axes {
		if e.Name == "" {
			continue
		}
		code, ok := linuxKeyMap[e.Name]
		if !ok {
			return errors.Errorf("Unrecognized axis name '%s'", e.Name)
		}
		setup.absmin[code] = -128
		setup.absmax[code] = 128
		setup.absflat[code] = 2
		setup.absfuzz[code] = 2
		err := o.ui_ioctl(C.UI_SET_ABSBIT, uintptr(code))
		if err != nil {
			return errors.Wrap(err, "ioctl uinput_setbit_abs")
		}
	}

	n, err := C.write_uinput_setup(&setup, C.int(o.fd))
	if err != nil {
		return errors.Wrap(err, "write uinput_user_dev")
	} else if n != C.sizeof_struct_uinput_user_dev {
		return errors.Errorf("Short write for uinput setup")
	}
	return nil
}

func NewUInput(m ControllerMapping, name string) (jcpc.Output, error) {
	o := &uinput{}

	fd, err := unix.Open("/dev/uinput", unix.O_RDWR|unix.O_NONBLOCK, 0)
	if err != nil {
		return nil, err
	}
	o.fd = fd

	err = o.ui_ioctl(C.UI_SET_EVBIT, C.EV_SYN)
	if err != nil {
		unix.Close(fd)
		return nil, errors.Wrap(err, "ioctl uinput_set_eventbit")
	}
	err = o.ui_ioctl(C.UI_SET_EVBIT, C.EV_ABS)
	if err != nil {
		unix.Close(fd)
		return nil, errors.Wrap(err, "ioctl uinput_set_eventbit")
	}
	err = o.ui_ioctl(C.UI_SET_EVBIT, C.EV_KEY)
	if err != nil {
		unix.Close(fd)
		return nil, errors.Wrap(err, "ioctl uinput_set_eventbit")
	}
	// TODO
	//err = o.ui_ioctl(C.UI_SET_EVBIT, C.EV_FF)
	if err != nil {
		unix.Close(fd)
		return nil, errors.Wrap(err, "ioctl uinput_set_eventbit")
	}

	var version_a C.uint

	version_b, err := o.ui_ioctl_r(C.UI_GET_VERSION, uintptr(unsafe.Pointer(&version_a)))
	if err == nil && (version_a == 5 || version_b == 5) {
		err = o.setupNewKernel(m, name)
	} else {
		fmt.Printf("Using old uinput interface from before kernel 4.5 (%d %d %v)\n", version_a, version_b, err)
		err = o.setupOldKernel(m, name)
	}
	if err != nil {
		unix.Close(fd)
		return nil, err
	}

	o.buttons = commonMappingToInternal(m)
	for _, code := range o.buttons.KeyCodes {
		if code == 0 {
			continue
		}

		err = o.ui_ioctl(C.UI_SET_KEYBIT, uintptr(code))
		if err != nil {
			unix.Close(fd)
			return nil, errors.Wrap(err, "ioctl uinput_set_keybit")
		}
	}

	// TODO force feedback setup

	err = o.ui_ioctl(C.UI_DEV_CREATE, 0)
	if err != nil {
		unix.Close(fd)
		return nil, errors.Wrap(err, "ioctl uinput_create_device")
	}

	return o, nil
}

func (o *uinput) BeginUpdate() error {
	o.mu.Lock()
	return nil
}

func (o *uinput) ButtonUpdate(b jcpc.ButtonID, state bool) {
	keyCode := o.buttons.KeyCodes[b.GetIndex()]
	if keyCode == 0 {
		return
	}
	val := int32(0)
	if state {
		val = 1
	}
	o.pending = append(o.pending, uinputEvent{
		Type:  C.EV_KEY,
		Code:  keyCode,
		Value: val,
	})
}

func (o *uinput) StickUpdate(axis jcpc.AxisID, value int8) {
	var code uint16
	var ok bool

	for _, e := range o.axes {
		if e.Axis == axis {
			code, ok = linuxKeyMap[e.Name]
			break
		}
	}
	if !ok {
		return
	}
	o.pending = append(o.pending, uinputEvent{
		Type:  C.EV_ABS,
		Code:  code,
		Value: int32(value),
	})
}

func (o *uinput) GyroUpdate(vals jcpc.GyroFrame) {}

func (o *uinput) FlushUpdate() error {
	defer o.mu.Unlock()

	buf := make([]byte, (1+len(o.pending))*C.sizeof_struct_input_event)
	for i, v := range o.pending {
		v.EncodeTo(buf[i*C.sizeof_struct_input_event:])
	}
	evSync := uinputEvent{
		Type:  C.EV_SYN,
		Code:  C.SYN_REPORT,
		Value: 0,
	}
	evSync.EncodeTo(buf[len(o.pending)*C.sizeof_struct_input_event:])
	n, err := unix.Write(o.fd, buf)
	if n != len(buf) {
		fmt.Println("[!!] short uinput write", n)
	}
	return err
}

func (o *uinput) OnFrame() {}

func (o *uinput) Close() error {
	o.mu.Lock()
	defer o.mu.Unlock()

	unix.Close(o.fd)
	o.fd = -1
	return nil
}
