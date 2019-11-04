package modbus

type lrc struct {
	sum uint8
}

func (sf *lrc) reset() *lrc {
	sf.sum = 0
	return sf
}

func (sf *lrc) push(data ...byte) *lrc {
	for _, b := range data {
		sf.sum += b
	}
	return sf
}

func (sf *lrc) value() byte {
	return uint8(-int8(sf.sum))
}
