package jcpc

type ButtonState [3]byte

type ButtonID int16
type AxisID int16

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
const (
	Button_Minus ButtonID = 0x100 + (1 << iota)
	Button_Plus
	Button_R_Stick
	Button_L_Stick
	Button_Home
	Button_Capture
	Button_Unused1
	Button_Unused2
)
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
	Button_Unused2,

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
	Button_R_Y:     "Y",
	Button_R_X:     "X",
	Button_R_B:     "B",
	Button_R_A:     "A",
	Button_R_SR:    "R-SR",
	Button_R_SL:    "R-SL",
	Button_R_R:     "R",
	Button_R_ZR:    "ZR",
	Button_Minus:   "-",
	Button_Plus:    "+",
	Button_R_Stick: "RStick",
	Button_L_Stick: "LStick",
	Button_Home:    "Home",
	Button_Capture: "Capture",
	Button_Unused1: "Unused1",
	Button_Unused2: "Unused2",
	Button_L_Down:  "Down",
	Button_L_Up:    "Up",
	Button_L_Right: "Right",
	Button_L_Left:  "Left",
	Button_L_SR:    "L-SR",
	Button_L_SL:    "L-SL",
	Button_L_L:     "L",
	Button_L_ZL:    "ZL",
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

func ButtonsFromSlice(b []byte) ButtonState {
	var result ButtonState
	result[0] = b[0]
	result[1] = b[1]
	result[2] = b[2]
	return result
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
