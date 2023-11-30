package event

type Event struct {
	Id                   string   `json:"id"`
	CreatedAt            int64    `json:"created_at"`
	Tags                 []string `json:"tags"`
	KeyId                string   `json:"key_id"`
	CostInUsd            float64  `json:"cost_in_usd"`
	Provider             string   `json:"provider"`
	Model                string   `json:"model"`
	Status               int      `json:"status"`
	PromptTokenCount     int      `json:"prompt_token_count"`
	CompletionTokenCount int      `json:"completion_token_count"`
	LatencyInMs          int      `json:"latency_in_ms"`
	Path                 string   `json:"path"`
	Method               string   `json:"method"`
	CustomId             string   `json:"custom_id"`
}
