package errors

type RateLimitError struct {
	message string
}

func NewRateLimitError(msg string) *RateLimitError {
	return &RateLimitError{
		message: msg,
	}
}

func (rle *RateLimitError) Error() string {
	return rle.message
}

func (rle *RateLimitError) RateLimit() {}
