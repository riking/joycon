package output

import "github.com/riking/joycon/prog4/jcpc"

type commonKeyMap struct {
	Button jcpc.ButtonID
	Name   string
}

type commonStickMap struct {
	Axis   jcpc.AxisID
	Invert bool
	Name   string
}

type ControllerMapping struct {
	Keys []commonKeyMap
	Axes []commonStickMap
}

// https://w3c.github.io/gamepad/#remapping
// JoyCons are negative left/down, positive up/right (docked orientation)
// System expects negative left/up, positive right/down
// screencheat expects positive up

var MappingL = ControllerMapping{
	Keys: []commonKeyMap{
		{jcpc.Button_L_Left, "GamepadSouth"},
		{jcpc.Button_L_Right, "GamepadNorth"},
		{jcpc.Button_L_Up, "GamepadWest"},
		{jcpc.Button_L_Down, "GamepadEast"},

		{jcpc.Button_L_SL, "GamepadTL"},
		{jcpc.Button_L_SR, "GamepadTR"},
		{jcpc.Button_L_L, "GamepadTL2"}, // TODO - what should the side buttons be mapped to
		{jcpc.Button_L_ZL, "GamepadTR2"},

		{jcpc.Button_Capture, "GamepadStart"},
		{jcpc.Button_Minus, "GamepadSelect"},
		{jcpc.Button_L_Stick, "GamepadLStick"},
	},
	Axes: []commonStickMap{
		{jcpc.Axis_L_Vertical, false, "MainStickHoriz"},
		{jcpc.Axis_L_Horiz, false, "MainStickVertical"},
	},
}

var MappingR = ControllerMapping{
	Keys: []commonKeyMap{
		{jcpc.Button_R_A, "GamepadSouth"},
		{jcpc.Button_R_Y, "GamepadNorth"},
		{jcpc.Button_R_B, "GamepadWest"},
		{jcpc.Button_R_X, "GamepadEast"},

		{jcpc.Button_R_SL, "GamepadTL"},
		{jcpc.Button_R_SR, "GamepadTR"},
		{jcpc.Button_R_R, "GamepadTL2"},
		{jcpc.Button_R_ZR, "GamepadTR2"},

		{jcpc.Button_Home, "GamepadStart"},
		{jcpc.Button_Plus, "GamepadSelect"},
		{jcpc.Button_R_Stick, "GamepadLStick"},
	},
	Axes: []commonStickMap{
		{jcpc.Axis_R_Vertical, true, "MainStickHoriz"},
		{jcpc.Axis_R_Horiz, true, "MainStickVertical"},
	},
}

var MappingDual = ControllerMapping{
	Keys: []commonKeyMap{
		{jcpc.Button_R_B, "GamepadSouth"},
		{jcpc.Button_R_X, "GamepadNorth"},
		{jcpc.Button_R_Y, "GamepadWest"},
		{jcpc.Button_R_A, "GamepadEast"},

		{jcpc.Button_L_Up, "GamepadD-Up"},
		{jcpc.Button_L_Down, "GamepadD-Down"},
		{jcpc.Button_L_Left, "GamepadD-Left"},
		{jcpc.Button_L_Right, "GamepadD-Right"},

		{jcpc.Button_L_L, "GamepadTL"},
		{jcpc.Button_L_ZL, "GamepadTL2"},
		{jcpc.Button_R_R, "GamepadTR"},
		{jcpc.Button_R_ZR, "GamepadTR2"},

		// TODO better mappings?
		{jcpc.Button_L_SL, "GamepadExtra1"},
		{jcpc.Button_L_SR, "GamepadExtra2"},
		{jcpc.Button_R_SL, "GamepadExtra3"},
		{jcpc.Button_R_SR, "GamepadExtra4"},

		{jcpc.Button_Home, "GamepadLogo"},
		{jcpc.Button_Capture, "GamepadCapture"},
		{jcpc.Button_Plus, "GamepadStart"},
		{jcpc.Button_Minus, "GamepadSelect"},
		{jcpc.Button_R_Stick, "GamepadRStick"},
		{jcpc.Button_L_Stick, "GamepadLStick"},
	},
	Axes: []commonStickMap{
		{jcpc.Axis_L_Horiz, true, "MainStickHoriz"},
		{jcpc.Axis_L_Vertical, false, "MainStickVertical"},
		{jcpc.Axis_R_Horiz, true, "SecondStickHoriz"},
		{jcpc.Axis_R_Vertical, false, "SecondStickVertical"},
	},
}

func RemapInputs(mappings *ControllerMapping, mods jcpc.InputRemappingOptions){
	for _,searched := range mods.InvertedAxes{
		for i,axis := range mappings.Axes{
			if axis.Axis == searched{
				mappings.Axes[i].Invert = !axis.Invert
			}
		}
	}
}
