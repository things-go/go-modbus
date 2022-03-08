package modbus

// LRC lrc sum.
type LRC struct {
	sum uint8
}

// Reset rest lrc sum.
func (sf *LRC) Reset() *LRC {
	sf.sum = 0
	return sf
}

// Push data in sum.
func (sf *LRC) Push(data ...byte) *LRC {
	for _, b := range data {
		sf.sum += b
	}
	return sf
}

// Value got lrc value.
func (sf *LRC) Value() byte {
	return uint8(-int8(sf.sum))
}
