package pixi

// Packs a boolean value into a bit channel byte array at the specified bit index. Used
// for boolean channels in separated mode only.
func PackBool(value bool, raw []byte, bitIndex int) {
	byteIndex := bitIndex / 8
	bitPos := bitIndex % 8
	if byteIndex >= len(raw) {
		return
	}
	if value {
		raw[byteIndex] |= 1 << bitPos
	} else {
		raw[byteIndex] &^= 1 << bitPos
	}
}

// Unpacks a boolean value from a bit channel byte array at the specified bit index. Used
// for boolean channels in separated mode only.
func UnpackBool(raw []byte, bitIndex int) bool {
	byteIndex := bitIndex / 8
	bitPos := bitIndex % 8
	if byteIndex >= len(raw) {
		return false
	}
	return (raw[byteIndex] & (1 << bitPos)) != 0
}
