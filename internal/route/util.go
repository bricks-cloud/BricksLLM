package route

import (
	"fmt"

	goopenai "github.com/sashabaranov/go-openai"
)

func ComputeCacheKeyForEmbeddingsRequest(path string, req *goopenai.EmbeddingRequest) string {
	if req == nil {
		return ""
	}

	input := ""
	if arr, ok := req.Input.([]interface{}); ok {
		for _, ele := range arr {
			converted, ok := ele.(string)
			if ok {
				input += converted
			}
		}
	} else if ele, ok := req.Input.(string); ok {
		input += ele
	}

	return fmt.Sprintf("%s-%s-%s-%s", path, input, req.EncodingFormat, req.User)
}

func ComputeCacheKeyForChatCompletionRequest(path string, req *goopenai.ChatCompletionRequest) string {
	if req == nil {
		return ""
	}

	input := ""
	for _, m := range req.Messages {
		input += m.Name
		input += m.Role
		input += m.Content
	}

	return fmt.Sprintf("%s-%s-%s-%s", path, input, req.User, req.ResponseFormat)
}
