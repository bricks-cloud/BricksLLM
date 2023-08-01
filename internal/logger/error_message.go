package logger

type ErrorMessage struct {
	Type       MessageType `json:"type"`
	InstanceId string      `json:"instanceId"`
	Message    string      `json:"message"`
	CreatedAt  int64       `json:"createdAt"`
}

func (em *ErrorMessage) DevLogContext() string {
	return "ERROR | "
}

func (em *ErrorMessage) SetInstanceId(instanceId string) {
	em.InstanceId = instanceId
}

func (em *ErrorMessage) SetMessage(message string) {
	em.Message = message
}

func (em *ErrorMessage) SetCreatedAt(createdAt int64) {
	em.CreatedAt = createdAt
}

func NewErrorMessage() *ErrorMessage {
	return &ErrorMessage{
		Type: ErrorMessageType,
	}
}
