package jcpc

import "sync"

func SetPlayerLights(jc JoyCon, pattern byte) {
	command := []byte{0x30, byte(pattern)}
	jc.SendCustomSubcommand(command)
}

func SetHomeLightPulse(jc JoyCon, pulseData []byte) {
	command := append([]byte{0x38}, pulseData...)
	jc.SendCustomSubcommand(command)
}

const SPIMaxData = 0x1C

func SPIFlashRead(jc JoyCon, addr, size uint32) ([]byte, error) {
	if size > SPIMaxData {
		return largeSPIRead(jc, addr, size)
	}

	var err error
	var b []byte
	for attempts := 0; attempts < 4; attempts++ {
		b, err = jc.SPIRead(addr, byte(size))
		if err != nil {
			continue
		}
		return b, nil
	}
	return nil, err
}

func SPIFlashWrite(jc JoyCon, addr uint16, p []byte) error {
	l := len(p)
	if l > SPIMaxData {
		l = SPIMaxData
	}

	panic("NotImplemented: need joycon to do a completion callback")
}

func largeSPIRead(jc JoyCon, addr, size uint32) ([]byte, error) {
	buf := make([]byte, size)

	const jobNum = 3
	type work struct {
		Off uint32
		Len byte
	}
	ch := make(chan work, 2)
	errCh := make(chan error, jobNum)
	var wg sync.WaitGroup

	wg.Add(jobNum)
	for i := 0; i < jobNum; i++ {
		go func(jn int) {
			defer wg.Done()
			var err error
			var b []byte
			for r := range ch {
				for attempts := 0; attempts < 4; attempts++ {
					b, err = jc.SPIRead(addr+r.Off, r.Len)
					if err != nil {
						continue
					}
					copy(buf[r.Off:], b)
					err = nil
					break
				}
				if err != nil {
					errCh <- err
					return
				}
			}
		}(i)
	}

	var err error
	for off := uint32(0); off < size; off += SPIMaxData {
		if off+SPIMaxData > size {
			select {
			case ch <- work{Off: off, Len: byte(size - off)}:
			case err = <-errCh:
				break
			}
		} else {
			select {
			case ch <- work{Off: off, Len: SPIMaxData}:
			case err = <-errCh:
				break
			}
		}
	}
	close(ch)
	wg.Wait()

	if err != nil {
		return nil, err
	}
	return buf, nil
}
