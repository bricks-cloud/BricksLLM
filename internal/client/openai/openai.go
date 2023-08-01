package openai

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/bricks-cloud/bricksllm/config"
	"github.com/bricks-cloud/bricksllm/internal/logger"
	"github.com/bricks-cloud/bricksllm/internal/util"
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

type OpenAiResponse struct {
	Id      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Choices []choice `json:"choices"`
	Usage   usage    `json:"usage"`
}

type OpenAiErrorResponse struct {
	Error *OpenAiErrorContent `json:"error"`
}

type OpenAiErrorContent struct {
	Message string `json:"message"`
	Type    string `json:"type"`
}

const (
	AuthorizationHeader string = "Authorization"
	ContentTypeHeader   string = "Content-Type"
)

func (c OpenAiClient) Send(rc *config.OpenAiRouteConfig, prompts []*config.OpenAiPrompt, lm *logger.LlmMessage) (*OpenAiResponse, error) {
	if len(c.apiCredential) == 0 && len(rc.ApiCredential) == 0 {
		return nil, errors.New("openai api credentials not found")
	}

	messages := []message{}
	loggerMessages := []logger.Message{}
	for _, prompt := range prompts {
		messages = append(messages, message{
			Role:    string(prompt.Role),
			Content: prompt.Content,
		})

		loggerMessages = append(loggerMessages, logger.Message{
			Role:    string(prompt.Role),
			Content: prompt.Content,
		})
	}
	lm.SetRequestMessages(loggerMessages)

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

	req.Header.Set(ContentTypeHeader, "application/json")
	req.Header.Set(AuthorizationHeader, fmt.Sprintf("Bearer %s", selected))

	lm.SetRequestHeaders(util.FilterHeaders(req.Header, []string{
		AuthorizationHeader,
	}))
	lm.SetRequestSize(req.ContentLength)
	lm.SetRequestCreatedAt(time.Now().Unix())

	res, err := c.httpClient.Do(req)
	lm.SetResponseCreatedAt(time.Now().Unix())
	if res != nil {
		lm.SetResponseStatus(res.StatusCode)
		lm.SetResponseHeaders(res.Header)
	}

	if err != nil {
		return nil, fmt.Errorf("error sending http requests: %w", err)
	}

	b, err = io.ReadAll(res.Body)
	lm.SetResponseBodySize(int64(len(b)))
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	if res.StatusCode != http.StatusOK {
		openAiErr := &OpenAiErrorResponse{}
		err = json.Unmarshal(b, openAiErr)
		if err != nil {
			return nil, fmt.Errorf("error unmarshaling open ai error response : %w", err)
		}

		if openAiErr.Error == nil {
			return nil, fmt.Errorf("cannot parse open ai error response : %w", err)
		}

		return nil, NewOpenAiError(openAiErr.Error.Message, openAiErr.Error.Type, res.StatusCode)
	}

	defer res.Body.Close()

	openaiRes := &OpenAiResponse{}
	err = json.Unmarshal(b, openaiRes)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling open ai response : %w", err)
	}

	lm.SetEstimatedCost(EstimateCost(string(rc.Model), openaiRes.Usage.PromptTokens, openaiRes.Usage.CompletionTokens))

	choices := []logger.Choice{}
	for _, choice := range openaiRes.Choices {
		choices = append(choices, logger.Choice{
			Role:         choice.Message.Role,
			Content:      choice.Message.Content,
			FinishReason: choice.FinishReason,
		})
	}
	lm.SetResponseChoices(choices)

	return openaiRes, err
}

func NewOpenAiClient(apiCredential string) OpenAiClient {
	return OpenAiClient{
		apiCredential: apiCredential,
		httpClient:    http.Client{},
	}
}
