package modbus

type lrc struct {
	sum uint8
}

func (this *lrc) reset() *lrc {
	this.sum = 0
	return this
}

func (this *lrc) push(data ...byte) *lrc {
	for _, b := range data {
		this.sum += b
	}
	return this
}

func (this *lrc) value() byte {
	return uint8(-int8(this.sum))
}
