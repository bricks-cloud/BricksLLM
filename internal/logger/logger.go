package logger

type Logger interface {
	Infow(msg string, keysAndValues ...interface{})
	Info(args ...interface{})
	Infof(template string, args ...interface{})
}

type MessageType string

const (
	LlmMessageType string = "llm"
	ApiMessageType string = "api"
)
