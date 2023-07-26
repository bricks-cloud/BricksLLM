package logger

type LlmResponse struct {
	Id        string    `json:"id"`
	Model     string    `json:"model"`
	CreatedAt int64     `json:"created_at"`
	Token     Token     `json:"token"`
	Size      int       `json:"size"`
	Status    int       `json:"status"`
	Choices   []*Choice `json:"choices"`
}

type Choice struct {
	Role         string `json:"role"`
	Content      string `json:"content"`
	FinishReason string `json:"finish_reason"`
}

type LlmRequest struct {
	Headers   map[string]string `json:"headers"`
	Model     string            `json:"model"`
	CreatedAt int64             `json:"created_at"`
}

type Token struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	Total            int `json:"total"`
}

type LlmMessage struct {
	InstanceId    string       `json:"instanceId"`
	Type          MessageType  `json:"type"`
	Response      *LlmResponse `json:"response"`
	Request       *LlmRequest  `json:"request"`
	Provider      string       `json:"provider"`
	EstimatedCost float64      `json:"estimated_cost"`
	CreatedAt     int64        `json:"created_at"`
	Latency       int          `json:"latency"`
}
