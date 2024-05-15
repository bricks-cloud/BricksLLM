package openai

type MessageRequest struct {
	Role     string         `json:"role"`
	Content  any            `json:"content"`
	FileIds  []string       `json:"file_ids,omitempty"` //nolint:revive // backwards-compatibility
	Metadata map[string]any `json:"metadata,omitempty"`
}
