package jcpc

func SetPlayerLights(jc JoyCon, pattern byte) {
	command := []byte{0x30, byte(pattern)}
	jc.SendCustomSubcommand(command)
}

func SetHomeLightPulse(jc JoyCon, pulseData []byte) {
	command := append([]byte{0x38}, pulseData...)
	jc.SendCustomSubcommand(command)
}

const SPIMaxData = 0x1B

func SPIFlashRead(jc JoyCon, addr uint16, p []byte) error {
	l := len(p)
	if l > SPIMaxData {
		l = SPIMaxData
	}

	panic("NotImplemented: need joycon to do a completion callback")
}

func SPIFlashWrite(jc JoyCon, addr uint16, p []byte) error {
	l := len(p)
	if l > SPIMaxData {
		l = SPIMaxData
	}

	panic("NotImplemented: need joycon to do a completion callback")
}
