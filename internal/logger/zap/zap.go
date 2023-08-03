package zap

import (
	"encoding/json"
	"time"

	"github.com/bricks-cloud/bricksllm/internal/logger"
	"github.com/fatih/color"
	"github.com/mattn/go-colorable"
	"go.uber.org/zap"
	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
)

type prependEncoder struct {
	// embed a zapcore encoder
	// this makes prependEncoder implement the interface without extra work
	zapcore.Encoder
	cfg zapcore.EncoderConfig
	// zap buffer pool
	pool buffer.Pool
}

func (e *prependEncoder) Clone() zapcore.Encoder {
	return &prependEncoder{
		// cloning the encoder with the base config
		Encoder: zapcore.NewConsoleEncoder(e.cfg),
		pool:    buffer.NewPool(),
		cfg:     e.cfg,
	}
}

// implementing only EncodeEntry
func (e *prependEncoder) EncodeEntry(entry zapcore.Entry, fields []zapcore.Field) (*buffer.Buffer, error) {
	// new log buffer
	buf := e.pool.Get()
	blue := color.New(color.BgBlue)
	red := color.New(color.BgRed)

	coloredPrefix := blue.Sprint("[BRICKSLLM]")
	if entry.Level != zap.InfoLevel {
		coloredPrefix = red.Sprint("[BRICKSLLM]")
	}

	buf.AppendString(coloredPrefix)
	buf.AppendString(" ")
	buf.AppendString(e.toAtalasPrefix(entry.Level))
	buf.AppendString(" | ")
	buf.AppendString(time.Now().Format(time.RFC3339))
	buf.AppendString(" | ")

	// calling the embedded encoder's EncodeEntry to keep the original encoding format
	consolebuf, err := e.Encoder.EncodeEntry(entry, fields)
	if err != nil {
		return nil, err
	}

	// just write the output into your own buffer
	_, err = buf.Write(consolebuf.Bytes())
	if err != nil {
		return nil, err
	}
	return buf, nil
}

func (e *prependEncoder) toAtalasPrefix(lvl zapcore.Level) string {
	switch lvl {
	case zapcore.DebugLevel:
		return "DEBUG"
	case zapcore.InfoLevel:
		return "INFO"
	case zapcore.FatalLevel:
		return "FATAL"
	}
	return ""
}

func NewLogger(mode string) logger.Logger {
	rawJSON := []byte(`{
		"level": "debug",
		"encoding": "json",
		"outputPaths": ["stdout"],
		"errorOutputPaths": ["stderr"],
		"encoderConfig": {
		  "messageKey": "message",
		  "levelKey": "level",
		  "levelEncoder": "lowercase"
		}
	  }`)

	var cfg zap.Config

	if err := json.Unmarshal(rawJSON, &cfg); err != nil {
		panic(err)
	}

	if mode == "production" {
		cfg.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
		return zap.Must(cfg.Build()).Sugar()
	}

	cfg.EncoderConfig.LevelKey = zapcore.OmitKey

	enc := &prependEncoder{
		Encoder: zapcore.NewConsoleEncoder(cfg.EncoderConfig),
		pool:    buffer.NewPool(),
		cfg:     cfg.EncoderConfig,
	}

	zapLogger := zap.New(zapcore.NewCore(
		enc,
		zapcore.AddSync(colorable.NewColorableStdout()),
		zapcore.DebugLevel,
	))

	return zapLogger.Sugar()
}
