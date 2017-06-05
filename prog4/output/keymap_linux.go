package output

/*
#include <linux/input.h>
*/
import "C"

type linuxKeyCode struct {
	Name  string
	Value uint16
}

var linuxKeyNames = []linuxKeyCode{
	// linux requires that BTN_SOUTH be used
	{"GamepadSouth", C.BTN_SOUTH},
	{"GamepadEast", C.BTN_EAST},
	{"GamepadNorth", C.BTN_NORTH},
	{"GamepadWest", C.BTN_WEST},
	{"GamepadC", C.BTN_C},
	{"GamepadZ", C.BTN_Z},
	{"GamepadTL", C.BTN_TL},
	{"GamepadTR", C.BTN_TR},
	{"GamepadTL2", C.BTN_TL2},
	{"GamepadTR2", C.BTN_TR2},
	{"GamepadSelect", C.BTN_SELECT},
	{"GamepadStart", C.BTN_START},
	{"GamepadLogo", C.BTN_MODE},
	{"GamepadLStick", C.BTN_THUMBL},
	{"GamepadRStick", C.BTN_THUMBR},
	{"GamepadD-Up", C.BTN_DPAD_UP},
	{"GamepadD-Down", C.BTN_DPAD_DOWN},
	{"GamepadD-Left", C.BTN_DPAD_LEFT},
	{"GamepadD-Right", C.BTN_DPAD_RIGHT},
	{"Screenshot", C.KEY_SYSRQ},
}

var linuxAxisNames = []linuxKeyCode{
	{"MainStickHoriz", C.ABS_X},
	{"MainStickVertical", C.ABS_Y},
	{"SecondStickHoriz", C.ABS_Z},
	{"SecondStickVertical", C.ABS_RX},
}

var linuxKeyMap = make(map[string]uint16)

func init() {
	for _, e := range linuxKeyNames {
		linuxKeyMap[e.Name] = e.Value
	}
	for _, e := range linuxAxisNames {
		linuxKeyMap[e.Name] = e.Value
	}
}

// TODO should this return errors
func commonMappingToInternal(m ControllerMapping) internalKeyCodeMapping {
	var r internalKeyCodeMapping

	for _, v := range m.Keys {
		if v.Name == "" {
			continue
		}
		code := linuxKeyMap[v.Name]
		if code == 0 {
			continue
		}

		i := v.Button.GetIndex()
		if i == -1 {
			continue
		}
		r.KeyCodes[i] = code
	}
	return r
}
