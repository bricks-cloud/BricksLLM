package policy

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	goopenai "github.com/sashabaranov/go-openai"
)

type Action string

const (
	Block          Action = "block"
	AllowButWarn   Action = "allow_but_warn"
	AllowButRedact Action = "allow_but_redact"
	Allow          Action = "allow"
)

type Type string

type CustomRule struct {
	Definition string `json:"definition"`
	Action     Action `json:"action"`
}

type RegularExpressionRule struct {
	Definition string `json:"definition"`
	Action     Action `json:"action"`
}

type Policy struct {
	Id                     string                   `json:"id"`
	NameRule               Action                   `json:"nameRule"`
	AddressRule            Action                   `json:"addressRule"`
	EmailRule              Action                   `json:"emailRule"`
	SsnRule                Action                   `json:"ssnRule"`
	PasswordRule           Action                   `json:"passwordRule"`
	RegularExpressionRules []*RegularExpressionRule `json:"regularExpressionRules"`
	CustomRules            []*CustomRule            `json:"customRules"`
}

type Request struct {
	Contents []string `json:"content"`
	Policy   *Policy  `json:"policy"`
}

func (p *Policy) Filter(client http.Client, input any) error {
	if p == nil {
		return nil
	}

	switch input.(type) {
	case *goopenai.EmbeddingRequest:
		converted := input.(*goopenai.EmbeddingRequest)
		if inputs, ok := converted.Input.([]interface{}); ok {
			inputsToInspect := []string{}

			for _, input := range inputs {
				stringified, ok := input.(string)
				if !ok {
					return errors.New("input is not string")
				}

				inputsToInspect = append(inputsToInspect, stringified)
			}

			updated, err := p.inspect(client, inputsToInspect)
			if err != nil {
				return nil
			}

			converted.Input = updated

		} else if input, ok := converted.Input.(string); ok {
			updated, err := p.inspect(client, []string{input})
			if err != nil {
				return nil
			}

			if len(updated) == 1 {
				converted.Input = updated[0]
			}
		}

		return nil
	case *goopenai.ChatCompletionRequest:
		converted := input.(*goopenai.ChatCompletionRequest)
		newMessages := []goopenai.ChatCompletionMessage{}

		contents := []string{}
		for _, message := range converted.Messages {
			contents = append(contents, message.Content)
		}

		updatedContents, err := p.inspect(client, contents)
		if err != nil {
			return nil
		}

		if len(updatedContents) != len(converted.Messages) {
			return errors.New("updated contents length not consistent with existing content length")
		}

		for index, c := range updatedContents {
			newMessages = append(newMessages, goopenai.ChatCompletionMessage{
				Content:      c,
				Role:         converted.Messages[index].Role,
				ToolCalls:    converted.Messages[index].ToolCalls,
				ToolCallID:   converted.Messages[index].ToolCallID,
				Name:         converted.Messages[index].Name,
				FunctionCall: converted.Messages[index].FunctionCall,
			})
		}

		converted.Messages = newMessages

		return nil
	}

	return nil
}

func (p *Policy) inspect(client http.Client, contents []string) ([]string, error) {
	data, err := json.Marshal(&Request{
		Contents: contents,
		Policy:   p,
	})

	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "", io.NopCloser(bytes.NewReader(data)))
	if err != nil {
		return nil, err
	}

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	parsed := []string{}
	err = json.Unmarshal(body, &parsed)
	if err != nil {
		return nil, err
	}

	return parsed, nil
}
