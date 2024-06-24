package route

import (
	"fmt"

	"github.com/bricks-cloud/bricksllm/internal/hasher"
	"github.com/bricks-cloud/bricksllm/internal/util"
	goopenai "github.com/sashabaranov/go-openai"
)

func ComputeCacheKeyForEmbeddingsRequest(path string, req *goopenai.EmbeddingRequest) string {
	if req == nil {
		return ""
	}

	input, _ := util.ConvertAnyToStr(req.Input)

	return hasher.Hash(fmt.Sprintf("%s-%s-%s-%s", path, input, req.EncodingFormat, req.User))
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

	if req.ResponseFormat != nil && len(req.ResponseFormat.Type) != 0 {
		return hasher.Hash(fmt.Sprintf("%s-%s-%s-%s", path, input, req.User, string(req.ResponseFormat.Type)))
	}

	return hasher.Hash(fmt.Sprintf("%s-%s-%s", path, input, req.User))
}
