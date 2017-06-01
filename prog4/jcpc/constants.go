package jcpc

const (
	VENDOR_NINTENDO           = 0x057e
	JOYCON_PRODUCT_L          = 0x2006
	JOYCON_PRODUCT_R          = 0x2007
	JOYCON_PRODUCT_PRO        = 0x2009
	JOYCON_PRODUCT_CHARGEGRIP = 0x200e
)

type JoyConType int

const (
	JCSideInvalid JoyConType = iota
	JCSideLeft
	JCSideRight
	JCSideBoth
)

func (t JoyConType) IsLeft() bool {
	return t == JCSideLeft || t == JCSideBoth
}

func (t JoyConType) IsRight() bool {
	return t == JCSideRight || t == JCSideBoth
}
