package logger

type Logger interface {
	Infow(msg string, keysAndValues ...interface{})
	Info(args ...interface{})
	Infof(template string, args ...interface{})
	Sync() error
	Debug(args ...interface{})
	Debugf(template string, args ...interface{})
	Debugw(template string, args ...interface{})
	Fatalf(template string, args ...interface{})
	Fatal(args ...interface{})
}

type MessageType string
