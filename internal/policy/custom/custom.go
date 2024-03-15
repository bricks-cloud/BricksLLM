package custompolicy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/sashabaranov/go-openai"
	goopenai "github.com/sashabaranov/go-openai"
)

type OpenAiDetector struct {
	client *goopenai.Client
	rt     time.Duration
}

func NewOpenAiDetector(rt time.Duration, key string) *OpenAiDetector {
	return &OpenAiDetector{
		client: goopenai.NewClient(key),
		rt:     rt,
	}
}

type result struct {
	RelevantTextsFound bool `json:"relevant_texts_found"`
}

func (c *OpenAiDetector) Detect(input []string, requirements []string) (bool, error) {
	requirement := ""
	for index, req := range requirements {
		requirement += fmt.Sprintf("%d.%s \n", index, req)
	}

	resp, err := c.client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT4Turbo0125,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: "You are a helpful assistant. You take in an array of strings and ouput JSON with one field called relevant_texts_found. relevant_texts_found is a boolean field that indicates whether or not given texts contain subtexts that fullfill the following requirements: " + requirement,
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: fmt.Sprintf("[%s]", strings.Join(input, " ,")),
				},
			},
			ResponseFormat: &goopenai.ChatCompletionResponseFormat{
				Type: goopenai.ChatCompletionResponseFormatTypeJSONObject,
			},
		},
	)

	if err != nil {
		return false, err
	}

	if len(resp.Choices) == 1 {
		r := &result{}
		err = json.Unmarshal([]byte(resp.Choices[0].Message.Content), r)
		if err != nil {
			return false, err
		}

		return r.RelevantTextsFound, nil
	}

	return false, errors.New("there are no choices from OpenAI")
}
