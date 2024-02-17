package errors

type CostLimitError struct {
	message string
}

func NewCostLimitError(msg string) *CostLimitError {
	return &CostLimitError{
		message: msg,
	}
}

func (cle *CostLimitError) Error() string {
	return cle.message
}

func (rle *CostLimitError) CostLimit() {}
