package errors

type ValidationError struct {
	message string
}

func NewValidationError(msg string) *ValidationError {
	return &ValidationError{
		message: msg,
	}
}

func (ve *ValidationError) Error() string {
	return ve.message
}

func (ve *ValidationError) Validation() {}
