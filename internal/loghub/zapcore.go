package loghub

import (
	"time"

	"go.uber.org/zap/zapcore"
)

// ServerLogEvent matches OpenAPI ServerLogEvent schema (subset used by server).
type ServerLogEvent struct {
	Timestamp time.Time      `json:"timestamp"`
	Level     string         `json:"level"`
	Logger    string         `json:"logger"`
	Message   string         `json:"message"`
	EventType string         `json:"eventType"`
	Fields    map[string]any `json:"fields,omitempty"`
}

// PiLogEvent matches OpenAPI PiLogEvent.
type PiLogEvent struct {
	Timestamp time.Time      `json:"timestamp"`
	Level     string         `json:"level"`
	Message   string         `json:"message"`
	Fields    map[string]any `json:"fields,omitempty"`
}

// ZapCore publishes zap entries to a Hub as ServerLogEvent JSON.
type ZapCore struct {
	hub *Hub
	lev zapcore.LevelEnabler
}

func NewZapCore(hub *Hub, lev zapcore.LevelEnabler) zapcore.Core {
	return &ZapCore{hub: hub, lev: lev}
}

func (c *ZapCore) Enabled(level zapcore.Level) bool {
	return c.lev.Enabled(level)
}

func (c *ZapCore) With([]zapcore.Field) zapcore.Core {
	return c
}

func (c *ZapCore) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(ent.Level) {
		return ce.AddCore(ent, c)
	}
	return ce
}

func (c *ZapCore) Write(ent zapcore.Entry, fields []zapcore.Field) error {
	menc := zapcore.NewMapObjectEncoder()
	for _, f := range fields {
		f.AddTo(menc)
	}
	var fm map[string]any
	if len(menc.Fields) > 0 {
		fm = menc.Fields
	}
	evt := ServerLogEvent{
		Timestamp: ent.Time.UTC(),
		Level:     ent.Level.String(),
		Logger:    ent.LoggerName,
		Message:   ent.Message,
		EventType: "application",
		Fields:    fm,
	}
	c.hub.Publish(evt)
	return nil
}

func (c *ZapCore) Sync() error {
	return nil
}
