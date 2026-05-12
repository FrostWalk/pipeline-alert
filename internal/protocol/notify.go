package protocol

const PlaySound byte = 0x01

func IsPlaySound(payload []byte) bool {
	return len(payload) == 1 && payload[0] == PlaySound
}
