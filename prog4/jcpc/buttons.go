package jcpc

import "encoding/binary"

type ButtonState [3]byte

type ButtonID int16
type AxisID int16

// First byte of ButtonState.
const (
	Button_R_Y ButtonID = 0x000 + (1 << iota)
	Button_R_X
	Button_R_B
	Button_R_A
	Button_R_SR
	Button_R_SL
	Button_R_R
	Button_R_ZR
)

// Middle byte of ButtonState.
const (
	Button_Minus ButtonID = 0x100 + (1 << iota)
	Button_Plus
	Button_R_Stick
	Button_L_Stick
	Button_Home
	Button_Capture
	Button_Unused1
	Button_IsChargeGrip
)

// Last byte of ButtonState.
const (
	Button_L_Down ButtonID = 0x200 + (1 << iota)
	Button_L_Up
	Button_L_Right
	Button_L_Left
	Button_L_SR
	Button_L_SL
	Button_L_L
	Button_L_ZL
)

// Bits in the 0x3F "push" mode treated as a little-endian int16.  See
// ConvertPushReport().
const (
	ButtonPushDown = 1 << iota
	ButtonPushRight
	ButtonPushLeft
	ButtonPushUp // remember to rotate these for L and R Joy-Cons
	ButtonPushSL
	ButtonPushSR
	_
	_
	ButtonPushMinus
	ButtonPushPlus
	ButtonPushLStick
	ButtonPushRStick
	ButtonPushHome
	ButtonPushCapture
	ButtonPushLR
	ButtonPushZLZR
)

// All button combinations considered to be a L+R press or part of one
var (
	// Side Joy-Con
	ButtonsSLSR_R = ButtonState{byte((Button_R_SL | Button_R_SR) & 0xFF), 0, 0}
	ButtonsSLSR_L = ButtonState{0, 0, byte((Button_L_SL | Button_L_SR) & 0xFF)}
	// Upright Pair
	ButtonsRZR = ButtonState{byte((Button_R_R | Button_R_ZR) & 0xFF), 0, 0}
	ButtonsLZL = ButtonState{0, 0, byte((Button_L_L | Button_L_ZL) & 0xFF)}
	// Pro Controller
	ButtonsLR   = ButtonState{byte(Button_R_R & 0xFF), 0, byte(Button_L_L & 0xFF)}
	ButtonsZLZR = ButtonState{byte(Button_R_ZR & 0xFF), 0, byte(Button_L_ZL & 0xFF)}

	ButtonsAnyLR = ButtonState{}.Union(ButtonsRZR).Union(ButtonsLZL).Union(ButtonsSLSR_L).Union(ButtonsSLSR_R)
)

var ButtonList = []ButtonID{
	Button_R_Y,
	Button_R_X,
	Button_R_B,
	Button_R_A,
	Button_R_SR,
	Button_R_SL,
	Button_R_R,
	Button_R_ZR,

	Button_Minus,
	Button_Plus,
	Button_R_Stick,
	Button_L_Stick,
	Button_Home,
	Button_Capture,
	Button_Unused1,
	Button_IsChargeGrip,

	Button_L_Down,
	Button_L_Up,
	Button_L_Right,
	Button_L_Left,
	Button_L_SR,
	Button_L_SL,
	Button_L_L,
	Button_L_ZL,
}

const allButtonsLeft2 = Button_L_Down | Button_L_Up |
	Button_L_Right | Button_L_Left |
	Button_L_SR | Button_L_SL |
	Button_L_L | Button_L_ZL
const allButtonsLeft1 = Button_Capture | Button_Minus | Button_L_Stick

const allButtonsRight0 = Button_R_Y | Button_R_X |
	Button_R_B | Button_R_A |
	Button_R_SR | Button_R_SL |
	Button_R_R | Button_R_ZR
const allButtonsRight1 = Button_Home | Button_Plus | Button_R_Stick

func (b ButtonID) GetIndex() int {
	q := b & 0xFF
	add := int((b & 0xF00) >> 5) // convert byte index to multiple of 8 - x >> 8 << 3
	for i, v := range ButtonList {
		if q == v {
			return i + add
		}
	}
	return -1
}

const (
	// Stick - uint8 from [0, 255]
	Axis_L_Horiz = iota
	Axis_L_Vertical
	Axis_R_Horiz
	Axis_R_Vertical
	// Gyro - int16 from [-0x10FF, +0x10FF]
	Axis_Yaw_X
	Axis_Yaw_Y
	Axis_Pitch_X
	Axis_Pitch_Y
	Axis_Roll_X
	Axis_Roll_Y

	Axis_Orientation_Min = Axis_Yaw_X
)

var buttonNameMap = map[ButtonID]string{
	Button_R_Y:          "Y",
	Button_R_X:          "X",
	Button_R_B:          "B",
	Button_R_A:          "A",
	Button_R_SR:         "R-SR",
	Button_R_SL:         "R-SL",
	Button_R_R:          "R",
	Button_R_ZR:         "ZR",
	Button_Minus:        "-",
	Button_Plus:         "+",
	Button_R_Stick:      "RStick",
	Button_L_Stick:      "LStick",
	Button_Home:         "Home",
	Button_Capture:      "Capture",
	Button_Unused1:      "Unused1",
	Button_IsChargeGrip: "Charging Grip",
	Button_L_Down:       "Down",
	Button_L_Up:         "Up",
	Button_L_Right:      "Right",
	Button_L_Left:       "Left",
	Button_L_SR:         "L-SR",
	Button_L_SL:         "L-SL",
	Button_L_L:          "L",
	Button_L_ZL:         "ZL",
}

func (b ButtonID) String() string {
	return buttonNameMap[b]
}

var axisNameMap = map[AxisID]string{
	Axis_L_Vertical: "Up/Down",
	Axis_L_Horiz:    "Left/Right",
	Axis_R_Vertical: "2nd Up/Down",
	Axis_R_Horiz:    "2nd Left/Right",
	Axis_Yaw_X:      "Yaw X",
	Axis_Yaw_Y:      "Yaw Y",
	Axis_Pitch_X:    "Pitch X",
	Axis_Pitch_Y:    "Pitch Y",
	Axis_Roll_X:     "Roll X",
	Axis_Roll_Y:     "Roll Y",
}

// ButtonsFromSlice copies the provided slice from a standard input report into
// a ButtonState.
func ButtonsFromSlice(b []byte) ButtonState {
	var result ButtonState
	result[0] = b[0]
	result[1] = b[1]
	result[2] = b[2]
	return result
}

// ConvertPushReport converts a 0x3F "push" button press report into a
// ButtonState.  Only for Left and Right Joy-Cons (Pro Controllers use a
// different format).
//
// The report ID (0x3F) should be removed from the 'buttons' slice.
func ConvertPushReport(side JoyConType, buttons []byte) ButtonState {
	var out ButtonState
	var data = binary.LittleEndian.Uint16(buttons[:2])
	if side == TypeLeft {
		if 0 != data&ButtonPushDown {
			out = out.Set(Button_L_Left, true)
		}
		if 0 != data&ButtonPushRight {
			out = out.Set(Button_L_Down, true)
		}
		if 0 != data&ButtonPushLeft {
			out = out.Set(Button_L_Up, true)
		}
		if 0 != data&ButtonPushUp {
			out = out.Set(Button_L_Right, true)
		}
		if 0 != data&ButtonPushSL {
			out = out.Set(Button_L_SL, true)
		}
		if 0 != data&ButtonPushSR {
			out = out.Set(Button_L_SR, true)
		}
		if 0 != data&ButtonPushMinus {
			out = out.Set(Button_Minus, true)
		}
		if 0 != data&ButtonPushLStick {
			out = out.Set(Button_L_Stick, true)
		}
		if 0 != data&ButtonPushCapture {
			out = out.Set(Button_Capture, true)
		}
		if 0 != data&ButtonPushLR {
			out = out.Set(Button_L_L, true)
		}
		if 0 != data&ButtonPushZLZR {
			out = out.Set(Button_L_ZL, true)
		}
	} else if side == TypeRight {
		if 0 != data&ButtonPushDown {
			out = out.Set(Button_R_A, true)
		}
		if 0 != data&ButtonPushRight {
			out = out.Set(Button_R_X, true)
		}
		if 0 != data&ButtonPushLeft {
			out = out.Set(Button_R_B, true)
		}
		if 0 != data&ButtonPushUp {
			out = out.Set(Button_R_Y, true)
		}
		if 0 != data&ButtonPushSL {
			out = out.Set(Button_R_SL, true)
		}
		if 0 != data&ButtonPushSR {
			out = out.Set(Button_R_SR, true)
		}
		if 0 != data&ButtonPushPlus {
			out = out.Set(Button_Plus, true)
		}
		if 0 != data&ButtonPushRStick {
			out = out.Set(Button_R_Stick, true)
		}
		if 0 != data&ButtonPushHome {
			out = out.Set(Button_Home, true)
		}
		if 0 != data&ButtonPushLR {
			out = out.Set(Button_R_R, true)
		}
		if 0 != data&ButtonPushZLZR {
			out = out.Set(Button_R_ZR, true)
		}
	} else {
		panic("bad Type passed to ConvertPushReport")
	}
	return out
}

// Get the state of a single ButtonID.
func (b ButtonState) Get(i ButtonID) bool {
	return b[(i&0x0300)>>8]&byte(i&0xFF) != 0
}

func (b ButtonState) Set(i ButtonID, state bool) ButtonState {
	b[(i&0x0300)>>8] &^= byte(i & 0xFF)
	if state {
		b[(i&0x0300)>>8] |= byte(i & 0xFF)
	}
	return b
}

// Union returns a ButtonState with all 'on' positions contained in either argument.
func (b ButtonState) Union(other ButtonState) ButtonState {
	var result ButtonState
	result[0] = b[0] | other[0]
	result[1] = b[1] | other[1]
	result[2] = b[2] | other[2]
	return result
}

// DiffMask returns a ButtonState with a '1' bit everywhere that this state differs from `other`.
func (b ButtonState) DiffMask(other ButtonState) ButtonState {
	var result ButtonState
	result[0] = b[0] ^ other[0]
	result[1] = b[1] ^ other[1]
	result[2] = b[2] ^ other[2]
	return result
}

func (b ButtonState) Remove(side JoyConType) ButtonState {
	var result ButtonState
	switch side {
	case TypeLeft:
		result[0] = b[0]
		result[1] = b[1] &^ byte(allButtonsLeft1&0xFF)
		result[2] = b[2] &^ byte(allButtonsLeft2&0xFF)
	case TypeRight:
		result[0] = b[0] &^ byte(allButtonsRight0&0xFF)
		result[1] = b[1] &^ byte(allButtonsRight1&0xFF)
		result[2] = b[2]
	case TypeBoth:
		result[0] = b[0] &^ byte(allButtonsRight0&0xFF)
		result[1] = b[1] &^ byte((allButtonsRight1&0xFF)|(allButtonsLeft1&0xFF))
		result[2] = b[2] &^ byte(allButtonsLeft2&0xFF)
	}
	return result
}

func (b ButtonState) HasAll(mask ButtonState) bool {
	var result = true
	result = result && (b[0]&mask[0]) == mask[0]
	result = result && (b[1]&mask[1]) == mask[1]
	result = result && (b[2]&mask[2]) == mask[2]
	return result
}

func (b ButtonState) HasAny(mask ButtonState) bool {
	var result = false
	result = result || (b[0]&mask[0]) != 0
	result = result || (b[1]&mask[1]) != 0
	result = result || (b[2]&mask[2]) != 0
	return result
}

// Check for a L+R press.  If this might be half of a double Joy-Con, the
// second return value (maybeDouble) will be true.
func (b ButtonState) PairCheckSelf() (selfPair bool, maybeDouble bool) {
	if b.HasAll(ButtonsSLSR_L) || b.HasAll(ButtonsSLSR_R) {
		return true, false
	}
	if b.HasAll(ButtonsLR) || b.HasAll(ButtonsZLZR) {
		return true, false
	}
	if b.HasAny(ButtonsLZL) || b.HasAny(ButtonsRZR) {
		return false, true
	}
	return false, false
}

// Check for a double Joy-Con L+R press.  Make sure not to call this if either
// of the two controllers is a Pro Controller.
func (b ButtonState) PairCheckDouble(b2 ButtonState) bool {
	if b.HasAny(ButtonsLZL) {
		return b2.HasAny(ButtonsRZR)
	} else if b.HasAny(ButtonsRZR) {
		return b2.HasAny(ButtonsLZL)
	}
	return false
}
