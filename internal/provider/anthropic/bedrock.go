package anthropic

type BedrockCompletionRequest struct {
	Prompt            string   `json:"prompt"`
	MaxTokensToSample int      `json:"max_tokens_to_sample"`
	StopSequences     []string `json:"stop_sequences,omitempty"`
	Temperature       float32  `json:"temperature,omitempty"`
	TopP              int      `json:"top_p,omitempty"`
	TopK              int      `json:"top_k,omitempty"`
}

type BedrockCompletionResponse struct {
	Completion string          `json:"completion"`
	StopReason string          `json:"stop_reason"`
	Model      string          `json:"model"`
	Metrics    *BedrockMetrics `json:"amazon-bedrock-invocationMetrics"`
}

type BedrockMessageRequest struct {
	AnthropicVersion string    `json:"anthropic_version"`
	Messages         []Message `json:"messages"`
	MaxTokens        int       `json:"max_tokens"`
	StopSequences    []string  `json:"stop_sequences,omitempty"`
	Temperature      float32   `json:"temperature,omitempty"`
	TopP             int       `json:"top_p,omitempty"`
	TopK             int       `json:"top_k,omitempty"`
	Metadata         *Metadata `json:"metadata,omitempty"`
}

type BedrockMessagesStopResponse struct {
	Type    string          `json:"type"`
	Metrics *BedrockMetrics `json:"amazon-bedrock-invocationMetrics"`
}

type BedrockMetrics struct {
	InputTokenCount   int `json:"inputTokenCount"`
	OutputTokenCount  int `json:"outputTokenCount"`
	InvocationLatency int `json:"invocationLatency"`
	FirstByteLatency  int `json:"firstByteLatency"`
}

type BedrockMessageType struct {
	Type string `json:"type"`
}
