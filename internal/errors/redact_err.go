package errors

type RedactError struct {
	message string
}

func NewRedactError(msg string) *RedactError {
	return &RedactError{
		message: msg,
	}
}

func (we *RedactError) Error() string {
	return we.message
}

func (we *RedactError) Redacted() {}
