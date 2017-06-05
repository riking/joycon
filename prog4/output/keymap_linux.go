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
	{"GamepadCapture", C.BTN_GAMEPAD + 15},

	{"GamepadExtra1", C.BTN_TRIGGER_HAPPY + 0},
	{"GamepadExtra2", C.BTN_TRIGGER_HAPPY + 1},
	{"GamepadExtra3", C.BTN_TRIGGER_HAPPY + 2},
	{"GamepadExtra4", C.BTN_TRIGGER_HAPPY + 3},
	{"GamepadExtra5", C.BTN_TRIGGER_HAPPY + 4},
	{"GamepadExtra6", C.BTN_TRIGGER_HAPPY + 5},
	{"GamepadExtra7", C.BTN_TRIGGER_HAPPY + 6},
	{"GamepadExtra8", C.BTN_TRIGGER_HAPPY + 7},
}

var linuxAxisNames = []linuxKeyCode{
	// have to avoid ABS_X and ABS_Y because X thinks it's a mouse despite BTN_GAMEPAD
	{"MainStickHoriz", C.ABS_Z},
	{"MainStickVertical", C.ABS_RX},
	{"SecondStickHoriz", C.ABS_RY},
	{"SecondStickVertical", C.ABS_RZ},
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
