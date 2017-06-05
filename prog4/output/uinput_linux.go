package output

import (
	"github.com/riking/joycon/prog4/jcpc"
	"golang.org/x/sys/unix"
)

/*
#include <linux/input.h>
#include <linux/uinput.h>
#include <stddef.h>

static const struct input_event sample_ev;
const size_t sizeof_input_event = sizeof(sample_ev);
const size_t offsetof_type = offsetof(struct input_event, type);
const size_t offsetof_code = offsetof(struct input_event, code);
const size_t offsetof_value = offsetof(struct input_event, value);

typedef struct input_event struct_input_event;
typedef struct input_id struct_input_id;
typedef struct input_absinfo struct_input_absinfo;

*/
import "C"

// gyro resolution is 4096 points/g because it's value of 4096 at rest
// To send gyro events, we need multiple event nodes

type uinput struct {
	fd      uintptr
	gyro_fd uintptr
}

type uinputEvent struct {
	Type  uint16
	Code  uint16
	Value int32
}

func (u uinputEvent) EncodeTo(p []byte) int {
	binary.LittleEndian.PutUint16(p[C.offsetof_type:], u.Type)
	binary.LittleEndian.PutUint16(p[C.offsetof_code:], u.Code)
	binary.LittleEndian.PutUint32(p[C.offsetof_value:], uint32(u.Value))

	return C.sizeof_input_event
}

type inputMapping struct {
}

func NewUInput(dual bool) (jcpc.Output, error) {
	fd, err := unix.Open("/dev/uinput", unix.O_RDWR|unix.O_NONBLOCK, 0)
	if err != nil {
		return err
	}

}
