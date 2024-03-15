package errors

type WarningError struct {
	message string
}

func NewWarningError(msg string) *WarningError {
	return &WarningError{
		message: msg,
	}
}

func (we *WarningError) Error() string {
	return we.message
}

func (we *WarningError) Warnings() {}
