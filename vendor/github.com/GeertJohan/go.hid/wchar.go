package hid

/*
#cgo darwin LDFLAGS: -liconv
#cgo windows LDFLAGS: -liconv
#include <stdlib.h>
#ifdef __APPLE__
#  define LIBICONV_PLUG 1
#endif
#include <iconv.h>
#include <wchar.h>

const size_t sizeof_wchar_t = sizeof(wchar_t);
*/
import "C"

import "unsafe"

// iconv charset strings
var (
	iconvCharsetWchar = C.CString("wchar_t//TRANSLIT")
	iconvCharsetChar  = C.CString("//TRANSLIT")
	iconvCharsetAscii = C.CString("ascii//TRANSLIT")
	iconvCharsetUtf8  = C.CString("utf-8//TRANSLIT")
)

func wcharToGoString(ws *C.wchar_t) (output string, err error) {
	if ws == nil {
		return "", nil
	}
	iconv, err := C.iconv_open(iconvCharsetUtf8, iconvCharsetWchar)
	if iconv == nil || err != nil {
		return "", wrapError{w: err, ctx: "Could not open iconv"}
	}
	defer C.iconv_close(iconv)

	wsLen := C.wcslen(ws)
	outBuf := C.calloc(wsLen, 4)
	defer C.free(outBuf)

	var inBytesLeft C.size_t = wsLen * C.sizeof_wchar_t
	var outBytesLeft C.size_t = wsLen * 4
	var inPtr *C.char = (*C.char)(unsafe.Pointer(ws))
	var outPtr *C.char = (*C.char)(outBuf)

	_, err = C.iconv(iconv, &inPtr, &inBytesLeft, &outPtr, &outBytesLeft)
	if err != nil {
		return "", err
	}

	out := C.GoString((*C.char)(outBuf))
	return out, nil
}
