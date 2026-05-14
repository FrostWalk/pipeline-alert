package piws

import (
	"encoding/base64"
	"testing"
)

func TestUploadSound(t *testing.T) {
	data := []byte("hello world")
	var msgs []any

	err := UploadSound(func(v any) error {
		msgs = append(msgs, v)
		return nil
	}, "horn.mp3", data, "abc123", true)
	if err != nil {
		t.Fatalf("UploadSound returned error: %v", err)
	}
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}

	start, ok := msgs[0].(SoundUploadStart)
	if !ok {
		t.Fatalf("first message type mismatch: %T", msgs[0])
	}
	if start.Type != TypeSoundUploadStart || start.FileName != "horn.mp3" || start.SizeBytes != int64(len(data)) || start.SHA256 != "abc123" || !start.IsDefault {
		t.Fatalf("unexpected start message: %#v", start)
	}

	chunk, ok := msgs[1].(SoundUploadChunk)
	if !ok {
		t.Fatalf("second message type mismatch: %T", msgs[1])
	}
	if chunk.Type != TypeSoundUploadChunk || chunk.FileName != "horn.mp3" || chunk.Offset != 0 {
		t.Fatalf("unexpected chunk metadata: %#v", chunk)
	}
	got, err := base64.StdEncoding.DecodeString(chunk.DataB64)
	if err != nil {
		t.Fatalf("failed to decode chunk: %v", err)
	}
	if string(got) != string(data) {
		t.Fatalf("chunk payload mismatch: got %q want %q", string(got), string(data))
	}
}
