package openai

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/bricks-cloud/atlas/config"
)

type OpenAiClient struct {
	httpClient    http.Client
	apiCredential string
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAiPayload struct {
	Model    string    `json:"model"`
	Messages []message `json:"messages"`
}

type usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type choice struct {
	Index        int     `json:"index"`
	Message      message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

type openAiResponse struct {
	Id      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Choices []choice `json:"choices"`
	Usage   usage    `json:"usage"`
}

func (c OpenAiClient) Send(rc *config.OpenAiRouteConfig, prompts []*config.OpenAiPrompt) (*openAiResponse, error) {
	if len(c.apiCredential) == 0 && len(rc.ApiCredential) == 0 {
		return nil, errors.New("openai api credentials not found")
	}

	messages := []message{}
	for _, prompt := range prompts {
		messages = append(messages, message{
			Role:    string(prompt.Role),
			Content: prompt.Content,
		})
	}

	p := openAiPayload{
		Model:    string(rc.Model),
		Messages: messages,
	}

	b, err := json.Marshal(p)
	bodyReader := bytes.NewReader(b)

	if err != nil {
		return nil, fmt.Errorf("error when marhsalling open ai json payload: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, "https://api.openai.com/v1/chat/completions", bodyReader)
	if err != nil {
		return nil, fmt.Errorf("error creating http request: %v", err)
	}

	selected := c.apiCredential
	if len(rc.ApiCredential) != 0 {
		selected = rc.ApiCredential
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", selected))

	res, err := c.httpClient.Do(req)
	b, err = ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %v", err)
	}

	defer res.Body.Close()

	openaiRes := &openAiResponse{}
	err = json.Unmarshal(b, openaiRes)

	return openaiRes, err
}

func NewOpenAiClient(apiCredential string) OpenAiClient {
	return OpenAiClient{
		apiCredential: apiCredential,
		httpClient:    http.Client{},
	}
}
