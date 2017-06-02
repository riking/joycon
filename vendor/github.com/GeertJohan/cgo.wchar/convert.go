package wchar

/*
#cgo darwin LDFLAGS: -liconv
#cgo windows LDFLAGS: -liconv
#include <stdlib.h>
#ifdef __APPLE__
#  define LIBICONV_PLUG 1
#endif
#include <iconv.h>
#include <wchar.h>
*/
import "C"

import (
	"encoding/binary"
	"fmt"
	"unsafe"
)

// iconv charset strings
var (
	iconvCharsetWchar = C.CString("wchar_t//TRANSLIT")
	iconvCharsetChar  = C.CString("//TRANSLIT")
	iconvCharsetAscii = C.CString("ascii//TRANSLIT")
	iconvCharsetUtf8  = C.CString("utf-8//TRANSLIT")
)

// iconv documentation:
// Use iconv. It seems to support conversion between char and wchar_t
// http://www.gnu.org/savannah-checkouts/gnu/libiconv/documentation/libiconv-1.13/iconv_open.3.html
// http://www.gnu.org/savannah-checkouts/gnu/libiconv/documentation/libiconv-1.13/iconv.3.html
// http://www.gnu.org/savannah-checkouts/gnu/libiconv/documentation/libiconv-1.13/iconv_close.3.html

// Internal helper function, wrapped by several other functions
func convertGoStringToWcharString(input string) (output WcharString, err error) {
	// quick return when input is an empty string
	if input == "" {
		return NewWcharString(0), nil
	}

	// open iconv
	iconv, errno := C.iconv_open(iconvCharsetWchar, iconvCharsetUtf8)
	if iconv == nil || errno != nil {
		return nil, fmt.Errorf("Could not open iconv instance: %s", errno)
	}
	defer C.iconv_close(iconv)

	// calculate bufferSizes in bytes for C
	bytesLeftInCSize := C.size_t(len([]byte(input))) // count exact amount of bytes from input
	bytesLeftOutCSize := C.size_t(len(input) * 4)    // wide char seems to be 4 bytes for every single- or multi-byte character. Not very sure though.

	// input for C. makes a copy using C malloc and therefore should be free'd.
	inputCString := C.CString(input)
	defer C.free(unsafe.Pointer(inputCString))

	// create output buffer
	outputChars := make([]int8, len(input)*4)

	// output for C
	outputCString := (*C.char)(unsafe.Pointer(&outputChars[0]))

	// call iconv for conversion of charsets, return on error
	_, errno = C.iconv(iconv, &inputCString, &bytesLeftInCSize, &outputCString, &bytesLeftOutCSize)
	if errno != nil {
		return nil, errno
	}

	// convert []int8 to WcharString
	// create WcharString with same length as input, and one extra position for the null terminator.
	output = make(WcharString, 0, len(input)+1)
	// create buff to convert each outputChar
	wcharAsByteAry := make([]byte, 4)
	// loop for as long as there are output chars
	for len(outputChars) >= 4 {
		// create 4 position byte slice
		wcharAsByteAry[0] = byte(outputChars[0])
		wcharAsByteAry[1] = byte(outputChars[1])
		wcharAsByteAry[2] = byte(outputChars[2])
		wcharAsByteAry[3] = byte(outputChars[3])
		// combine 4 position byte slice into uint32
		wcharAsUint32 := binary.LittleEndian.Uint32(wcharAsByteAry)
		// find null terminator (doing this right?)
		if wcharAsUint32 == 0x0 {
			break
		}
		// append uint32 to outputUint32
		output = append(output, Wchar(wcharAsUint32))
		// reslice the outputChars
		outputChars = outputChars[4:]
	}
	// Add null terminator
	output = append(output, Wchar(0x0))

	return output, nil
}

// Internal helper function, wrapped by several other functions
func convertWcharStringToGoString(ws WcharString) (output string, err error) {
	// return empty string if len(input) == 0
	if len(ws) == 0 {
		return "", nil
	}

	// open iconv
	iconv, errno := C.iconv_open(iconvCharsetUtf8, iconvCharsetWchar)
	if iconv == nil || errno != nil {
		return "", fmt.Errorf("Could not open iconv instance: %s", errno.Error())
	}
	defer C.iconv_close(iconv)

	inputAsCChars := make([]C.char, 0, len(ws)*4)
	wcharAsBytes := make([]byte, 4)
	for _, nextWchar := range ws {
		// find null terminator
		if nextWchar == 0 {
			// Return empty string if there are no chars in buffer
			//++ FIXME: this should NEVER be the case because input is checked at the begin of this function.
			if len(inputAsCChars) == 0 {
				return "", nil
			}
			break
		}

		// split Wchar into bytes
		binary.LittleEndian.PutUint32(wcharAsBytes, uint32(nextWchar))

		// append the bytes as C.char to inputAsCChars
		for i := 0; i < 4; i++ {
			inputAsCChars = append(inputAsCChars, C.char(wcharAsBytes[i]))
		}
	}

	// input for C
	inputAsCCharsPtr := &inputAsCChars[0]

	// calculate buffer size for input
	bytesLeftInCSize := C.size_t(len(inputAsCChars))

	// calculate buffer size for output
	bytesLeftOutCSize := C.size_t(len(inputAsCChars))

	// create output buffer
	outputChars := make([]C.char, bytesLeftOutCSize)

	// output buffer pointer for C
	outputCharsPtr := &outputChars[0]

	// call iconv for conversion of charsets, return on error
	_, errno = C.iconv(iconv, &inputAsCCharsPtr, &bytesLeftInCSize, &outputCharsPtr, &bytesLeftOutCSize)
	if errno != nil {
		return "", errno
	}

	// conver output buffer to go string
	output = C.GoString((*C.char)(&outputChars[0]))

	return output, nil
}

// Internal helper function, wrapped by other functions
func convertGoRuneToWchar(r rune) (output Wchar, err error) {
	// quick return when input is an empty string
	if r == '\000' {
		return Wchar(0), nil
	}

	// open iconv
	iconv, errno := C.iconv_open(iconvCharsetWchar, iconvCharsetUtf8)
	if iconv == nil || errno != nil {
		return Wchar(0), fmt.Errorf("Could not open iconv instance: %s", errno)
	}
	defer C.iconv_close(iconv)

	// bufferSizes for C
	bytesLeftInCSize := C.size_t(4)
	bytesLeftOutCSize := C.size_t(4 * 4)
	// TODO/FIXME: the last 4 bytes as indicated by bytesLeftOutCSize wont be used...
	// iconv assumes each given char to be one wchar.
	// in this case we know that the given 4 chars will actually be one unicode-point and therefore will result in one wchar.
	// hence, we give the iconv library a buffer of 4 char's size, and tell the library that it has a buffer of 32 char's size.
	// if the rune would actually contain 2 unicode-point's this will result in massive failure (and probably the end of a process' life)

	// input for C. makes a copy using C malloc and therefore should be free'd.
	runeCString := C.CString(string(r))
	defer C.free(unsafe.Pointer(runeCString))

	// create output buffer
	outputChars := make([]C.char, 4)

	// output buffer pointer for C
	outputCharsPtr := &outputChars[0]

	// call iconv for conversion of charsets
	_, errno = C.iconv(iconv, &runeCString, &bytesLeftInCSize, &outputCharsPtr, &bytesLeftOutCSize)
	if errno != nil {
		return '\000', errno
	}

	// convert C.char's to Wchar
	wcharAsByteAry := make([]byte, 4)
	wcharAsByteAry[0] = byte(outputChars[0])
	wcharAsByteAry[1] = byte(outputChars[1])
	wcharAsByteAry[2] = byte(outputChars[2])
	wcharAsByteAry[3] = byte(outputChars[3])

	// combine 4 position byte slice into uint32 and convert to Wchar.
	wcharAsUint32 := binary.LittleEndian.Uint32(wcharAsByteAry)
	output = Wchar(wcharAsUint32)

	return output, nil
}

// Internal helper function, wrapped by several other functions
func convertWcharToGoRune(w Wchar) (output rune, err error) {
	// return  if len(input) == 0
	if w == 0 {
		return '\000', nil
	}

	// open iconv
	iconv, errno := C.iconv_open(iconvCharsetUtf8, iconvCharsetWchar)
	if iconv == nil || errno != nil {
		return '\000', fmt.Errorf("Could not open iconv instance: %s", errno.Error())
	}
	defer C.iconv_close(iconv)

	// split Wchar into bytes
	wcharAsBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(wcharAsBytes, uint32(w))

	// place the wcharAsBytes into wcharAsCChars
	// TODO: use unsafe.Pointer here to do the conversion?
	wcharAsCChars := make([]C.char, 0, 4)
	for i := 0; i < 4; i++ {
		wcharAsCChars = append(wcharAsCChars, C.char(wcharAsBytes[i]))
	}

	// pointer to the first wcharAsCChars
	wcharAsCCharsPtr := &wcharAsCChars[0]

	// calculate buffer size for input
	bytesLeftInCSize := C.size_t(4)

	// calculate buffer size for output
	bytesLeftOutCSize := C.size_t(4)

	// create output buffer
	outputChars := make([]C.char, 4)

	// output buffer pointer for C
	outputCharsPtr := &outputChars[0]

	// call iconv for conversion of charsets
	_, errno = C.iconv(iconv, &wcharAsCCharsPtr, &bytesLeftInCSize, &outputCharsPtr, &bytesLeftOutCSize)
	if errno != nil {
		return '\000', errno
	}

	// convert outputChars ([]int8, len 4) to Wchar
	// TODO: can this conversion be done easier by using this: ?
	// output = *((*rune)(unsafe.Pointer(&outputChars[0])))
	runeAsByteAry := make([]byte, 4)
	runeAsByteAry[0] = byte(outputChars[0])
	runeAsByteAry[1] = byte(outputChars[1])
	runeAsByteAry[2] = byte(outputChars[2])
	runeAsByteAry[3] = byte(outputChars[3])

	// combine 4 position byte slice into uint32 and convert to rune.
	runeAsUint32 := binary.LittleEndian.Uint32(runeAsByteAry)
	output = rune(runeAsUint32)

	return output, nil
}
