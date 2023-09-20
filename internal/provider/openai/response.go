package openai

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type ResponseMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Choice struct {
	Index        int             `json:"index"`
	Message      ResponseMessage `json:"message"`
	FinishReason string          `json:"finish_reason"`
}

type ChatCompletionResponse struct {
	Id      string   `json:"id"`
	Object  string   `json:"object"`
	Model   string   `json:"model"`
	Created int64    `json:"created"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

type ErrorContent struct {
	Message string  `json:"message"`
	Type    string  `json:"type,omitempty"`
	Param   *string `json:"param,omitempty"`
	Code    any     `json:"code,omitempty"`
}

type ChatCompletionErrorResponse struct {
	Error *ErrorContent `json:"error"`
}
