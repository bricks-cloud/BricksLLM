package logger

import "fmt"

type LlmResponse struct {
	Id        string              `json:"id"`
	Headers   map[string][]string `json:"headers"`
	CreatedAt int64               `json:"created_at"`
	Size      int64               `json:"size"`
	Status    int                 `json:"status"`
	Choices   []Choice            `json:"choices"`
}

type Choice struct {
	Role         string `json:"role"`
	Content      string `json:"content"`
	FinishReason string `json:"finish_reason"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type LlmRequest struct {
	Headers   map[string][]string `json:"headers"`
	Model     string              `json:"model"`
	Messages  []Message           `json:"messages"`
	Size      int64               `json:"size"`
	CreatedAt int64               `json:"created_at"`
}

type Token struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	Total            int `json:"total"`
}

type LlmMessage struct {
	InstanceId    string       `json:"instanceId"`
	Type          MessageType  `json:"type"`
	Token         *Token       `json:"token"`
	Response      *LlmResponse `json:"response"`
	Request       *LlmRequest  `json:"request"`
	Provider      string       `json:"provider"`
	EstimatedCost float64      `json:"estimated_cost"`
	CreatedAt     int64        `json:"created_at"`
	Latency       int64        `json:"latency"`
}

func NewLlmMessage() *LlmMessage {
	return &LlmMessage{
		Type:     LlmMessageType,
		Token:    &Token{},
		Response: &LlmResponse{},
		Request:  &LlmRequest{},
	}
}

func (lm *LlmMessage) DevLogContext() string {
	result := "LLM | "

	if lm.Response.Status != 0 {
		result += (colorStatusCode(lm.Response.Status) + " |")
	}

	if lm.Latency != 0 {
		result += fmt.Sprintf(" %dms |", lm.Latency)
	}

	if lm.Token.Total != 0 {
		result += fmt.Sprintf(" %d tokens |", lm.Token.Total)
	}

	return result
}

func (lm *LlmMessage) SetResponseId(id string) {
	lm.Response.Id = id
}

func (lm *LlmMessage) SetResponseCreatedAt(createdAt int64) {
	lm.Response.CreatedAt = createdAt
}

func (lm *LlmMessage) SetResponseHeaders(headers map[string][]string) {
	lm.Response.Headers = headers
}

func (lm *LlmMessage) SetResponseBodySize(size int64) {
	lm.Response.Size = size
}

func (lm *LlmMessage) SetResponseStatus(status int) {
	lm.Response.Status = status
}

func (lm *LlmMessage) SetResponseChoices(choices []Choice) {
	lm.Response.Choices = choices
}

func (lm *LlmMessage) SetRequestHeaders(headers map[string][]string) {
	lm.Request.Headers = headers
}

func (lm *LlmMessage) SetRequestModel(model string) {
	lm.Request.Model = model
}

func (lm *LlmMessage) SetRequestSize(size int64) {
	lm.Request.Size = size
}

func (lm *LlmMessage) SetRequestCreatedAt(createdAt int64) {
	lm.Request.CreatedAt = createdAt
}

func (lm *LlmMessage) SetPromptTokens(tokens int) {
	lm.Token.PromptTokens = tokens
}

func (lm *LlmMessage) SetCompletionTokens(tokens int) {
	lm.Token.CompletionTokens = tokens
}

func (lm *LlmMessage) SetTotalTokens(tokens int) {
	lm.Token.Total = tokens
}

func (lm *LlmMessage) SetInstanceId(id string) {
	lm.InstanceId = id
}

func (lm *LlmMessage) SetProvider(provider string) {
	lm.Provider = provider
}

func (lm *LlmMessage) SetEstimatedCost(cost float64) {
	lm.EstimatedCost = cost
}

func (lm *LlmMessage) SetCreatedAt(createdAt int64) {
	lm.CreatedAt = createdAt
}

func (lm *LlmMessage) SetLatency(latency int64) {
	lm.Latency = latency
}

func (lm *LlmMessage) SetRequestMessages(messages []Message) {
	lm.Request.Messages = messages
}

type llmLoggerConfig interface {
	GetHideHeaders() bool
	GetHideResponseContent() bool
	GetHidePromptContent() bool
}

func (lm *LlmMessage) ModifyFileds(c llmLoggerConfig) {
	if c.GetHideResponseContent() {
		lm.Response.Choices = []Choice{}
	}

	if c.GetHidePromptContent() {
		lm.Request.Messages = []Message{}
	}

	if c.GetHideHeaders() {
		lm.Request.Headers = map[string][]string{}
		lm.Response.Headers = map[string][]string{}
	}
}
