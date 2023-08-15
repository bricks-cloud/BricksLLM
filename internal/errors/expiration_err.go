package errors

const (
	TtlExpiration       string = "ttl"
	CostLimitExpiration string = "cost-limit"
)

type ExpirationError struct {
	message string
	reason  string
}

func NewExpirationError(msg string, reason string) *ExpirationError {
	return &ExpirationError{
		message: msg,
		reason:  reason,
	}
}

func (ee *ExpirationError) Error() string {
	return ee.message
}

func (ee *ExpirationError) Reason() string {
	return ee.reason
}
