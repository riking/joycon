package jcpc

type ButtonState [3]byte

type ButtonID int16

const (
	Button_R_Y ButtonID = 0x000 + (1 << iota)
	Button_R_X
	Button_R_B
	Button_R_A
	Button_R_SR
	Button_R_SL
	Button_R_R
	Button_R_ZR

	Button_Minus ButtonID = 0x100 + (1 << iota)
	Button_Plus
	Button_R_Stick
	Button_L_Stick
	Button_Home
	Button_Capture

	Button_L_Down ButtonID = 0x200 + (1 << iota)
	Button_L_Up
	Button_L_Right
	Button_L_Left
	Button_L_SR
	Button_L_SL
	Button_L_L
	Button_L_ZL
)

func ButtonsFromSlice(b []byte) ButtonState {
	var result ButtonState
	result[0] = b[0]
	result[1] = b[1]
	result[2] = b[2]
	return result
}

// Get the state of a single ButtonID.
func (b ButtonState) Get(i ButtonID) bool {
	return b[(i & 0x0300) >> 8] & byte(i & 0xFF) != 0
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