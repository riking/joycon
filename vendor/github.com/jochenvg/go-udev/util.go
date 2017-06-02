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

import "unsafe"

func freeCharPtr(s *C.char) {
	C.free(unsafe.Pointer(s))
}
