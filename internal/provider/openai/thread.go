package openai

import (
	goopenai "github.com/sashabaranov/go-openai"
)

type ThreadRequest struct {
	Messages []ThreadMessage `json:"messages,omitempty"`
	Metadata map[string]any  `json:"metadata,omitempty"`
}

type ThreadMessageRole string

const (
	ThreadMessageRoleUser      ThreadMessageRole = "user"
	ThreadMessageRoleAssistant ThreadMessageRole = "assistant"
)

type ThreadMessage struct {
	Role     ThreadMessageRole `json:"role"`
	Content  any               `json:"content"`
	FileIDs  []string          `json:"file_ids,omitempty"`
	Metadata map[string]any    `json:"metadata,omitempty"`
}

type CreateThreadAndRunRequest struct {
	*goopenai.RunRequest
	Thread ThreadRequest `json:"thread"`
}

type ImageFileContentPart struct {
	Type      string    `json:"type"`
	ImageFile ImageFile `json:"image_file"`
}

type ImageUrlContentPart struct {
	Type     string   `json:"type"`
	ImageUrl ImageUrl `json:"image_url"`
}

type TextContentPart struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type ImageUrl struct {
	Url    string `json:"url"`
	Detail string `json:"detail"`
}

type ImageFile struct {
	FileId string `json:"file_id"`
	Detail string `json:"detail"`
}
