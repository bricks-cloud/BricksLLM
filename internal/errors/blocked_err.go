package errors

type BlockedError struct {
	message string
}

func NewBlockedError(msg string) *BlockedError {
	return &BlockedError{
		message: msg,
	}
}

func (be *BlockedError) Error() string {
	return be.message
}

func (be *BlockedError) Blocked() {}
