package piws

// Downlink message types (server -> Raspberry Pi client).

const (
	TypeSoundSyncStart = "soundSyncStart"
	TypeSoundSyncChunk = "soundSyncChunk"
	TypeSetActiveSound = "setActiveSound"
)

type SoundSyncStart struct {
	Type      string `json:"type"`
	FileName  string `json:"fileName"`
	SizeBytes int64  `json:"sizeBytes"`
}

type SoundSyncChunk struct {
	Type     string `json:"type"`
	FileName string `json:"fileName"`
	Offset   int64  `json:"offset"`
	DataB64  string `json:"dataBase64"`
}

type SetActiveSound struct {
	Type     string `json:"type"`
	FileName string `json:"fileName"`
}

// Uplink from Pi client.

const TypePiLog = "piLog"

type PiLog struct {
	Type    string         `json:"type"`
	Level   string         `json:"level"`
	Message string         `json:"message"`
	Fields  map[string]any `json:"fields,omitempty"`
}
