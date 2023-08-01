package openai

type OpenAiError struct {
	message   string
	errorType string
	code      int
}

func NewOpenAiError(message string, errorType string, code int) *OpenAiError {
	return &OpenAiError{
		message:   message,
		errorType: errorType,
		code:      code,
	}
}

func (e *OpenAiError) Error() string {
	return e.message
}

func (e *OpenAiError) StatusCode() int {
	return e.code
}
