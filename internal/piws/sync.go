package piws

import (
	"encoding/base64"
)

// ChunkRawBytes is raw sound bytes per websocket JSON message (before base64).
const ChunkRawBytes = 24 * 1024

// SendSound streams file content as JSON chunk messages via send callback.
func SendSound(send func(any) error, fileName string, data []byte) error {
	if err := send(SoundSyncStart{
		Type:      TypeSoundSyncStart,
		FileName:  fileName,
		SizeBytes: int64(len(data)),
	}); err != nil {
		return err
	}
	for offset := 0; offset < len(data); offset += ChunkRawBytes {
		end := offset + ChunkRawBytes
		if end > len(data) {
			end = len(data)
		}
		chunk := data[offset:end]
		msg := SoundSyncChunk{
			Type:     TypeSoundSyncChunk,
			FileName: fileName,
			Offset:   int64(offset),
			DataB64:  base64.StdEncoding.EncodeToString(chunk),
		}
		if err := send(msg); err != nil {
			return err
		}
	}
	return nil
}
