// +build linux,cgo

package udev

/*
  #cgo LDFLAGS: -ludev
  #include <libudev.h>
  #include <linux/types.h>
  #include <stdlib.h>
	#include <linux/kdev_t.h>

  int go_udev_major(dev_t d) {
    return MAJOR(d);
  }
  int go_udev_minor(dev_t d) {
    return MINOR(d);
  }
  dev_t go_udev_mkdev(int major, int minor) {
    return MKDEV(major, minor);
  }
*/
import "C"

// Devnum is a kernel device number
type Devnum struct {
	d C.dev_t
}

// Major returns the major part of a Devnum
func (d Devnum) Major() int {
	return int(C.go_udev_major(d.d))
}

// Minor returns the minor part of a Devnum
func (d Devnum) Minor() int {
	return int(C.go_udev_minor(d.d))
}

// MkDev creates a Devnum from a major and minor number
func MkDev(major, minor int) Devnum {
	return Devnum{C.go_udev_mkdev((C.int)(major), (C.int)(minor))}
}
