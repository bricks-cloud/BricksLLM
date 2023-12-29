package errors

type AuthError struct {
	message string
}

func NewAuthError(msg string) *AuthError {
	return &AuthError{
		message: msg,
	}
}

func (ae *AuthError) Error() string {
	return ae.message
}

func (ae *AuthError) Authenticated() {}
