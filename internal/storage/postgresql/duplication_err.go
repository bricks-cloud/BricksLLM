package postgresql

type DuplicationError struct {
	message string
}

func NewDuplicationError(msg string) *DuplicationError {
	return &DuplicationError{
		message: msg,
	}
}

func (de *DuplicationError) Error() string {
	return de.message
}

func (de *DuplicationError) Duplication() {}
