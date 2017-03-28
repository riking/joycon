
#include "joycon.h"
#include <wchar.h>

// R horiz: range -65 to 90
//          dead 0-15
// R vert:  range -82 to 52
//          dead -17 to -12
// L horiz: range -65 to 63
//          dead -17 to 3
// L vert:  range -57 to 77
//          dead 7-12
// calibrations.txt
// 98:b6:e9:74:1b:22 -64 -1 16 90 -81 -18 -11 51
// 98:b6:e9:34:d5:c2 -65 -17 3 63 -57 7 12 77

calibration_data calibration_file_load(wchar_t *serial) {
	calibration_data data;

	if (0 == wcscmp(serial, L"98:b6:e9:74:1b:22")) {
		data.horizontal =
		    (stick_calibration){0, 0x80 + -64, 0x80 + -1, 0x80 + 16, 0x80 + 90};
		data.vertical = (stick_calibration){0, 0x80 + -81, 0x80 + -18,
		                                    0x80 + -11, 0x80 + 51};
		return data;
	} else if (0 == wcscmp(serial, L"98:b6:e9:34:d5:c2")) {
		data.horizontal =
		    (stick_calibration){0, 0x80 + -65, 0x80 + -17, 0x80 + 3, 0x80 + 63};
		data.vertical =
		    (stick_calibration){0, 0x80 + -57, 0x80 + 7, 0x80 + 12, 0x80 + 77};
		return data;
	}
	// just guess
	data.horizontal =
	    (stick_calibration){1, 0x80 + -50, 0x80 + -15, 0x80 + 15, 0x80 + 50};
	data.vertical =
	    (stick_calibration){1, 0x80 + -50, 0x80 + -15, 0x80 + 15, 0x80 + 50};
	return data;
}