package policy

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	internal_errors "github.com/bricks-cloud/bricksllm/internal/errors"

	goopenai "github.com/sashabaranov/go-openai"
)

type Action string

const (
	Block          Action = "block"
	AllowButWarn   Action = "allow_but_warn"
	AllowButRedact Action = "allow_but_redact"
	Allow          Action = "allow"
)

type Rule string

const (
	Name    Rule = "name"
	Address Rule = "address"
	Email   Rule = "email"
	Ssn     Rule = "ssn"
)

type CustomRule struct {
	Definition string `json:"definition"`
	Action     Action `json:"action"`
}

type RegularExpressionRule struct {
	Definition string `json:"definition"`
	Action     Action `json:"action"`
}

type Config struct {
	Rules map[Rule]Action `json:"rules"`
}

type RegexConfig struct {
	RegularExpressionRules []*RegularExpressionRule `json:"rules"`
}

type CustomConfig struct {
	CustomRules []*CustomRule `json:"rules"`
}

type Policy struct {
	Id                     string                   `json:"id"`
	CreatedAt              int64                    `json:"createdAt"`
	UpdatedAt              int64                    `json:"updatedAt"`
	Tags                   []string                 `json:"tags"`
	Config                 *Config                  `json:"config"`
	RegexConfig            *RegexConfig             `json:"regexConfig"`
	CustomConfig           *CustomConfig            `json:"customConfig"`
	NameRule               Action                   `json:"nameRule"`
	AddressRule            Action                   `json:"addressRule"`
	EmailRule              Action                   `json:"emailRule"`
	SsnRule                Action                   `json:"ssnRule"`
	PasswordRule           Action                   `json:"passwordRule"`
	RegularExpressionRules []*RegularExpressionRule `json:"regularExpressionRules"`
	CustomRules            []*CustomRule            `json:"customRules"`
}

type Request struct {
	Contents []string `json:"contents"`
	Policy   *Policy  `json:"policy"`
}

type Response struct {
	Contents       []string        `json:"contents"`
	Action         Action          `json:"action"`
	Warnings       map[string]bool `json:"warnings"`
	BlockedReasons map[string]bool `json:"blockedReasons"`
}

func (p *Policy) Filter(client http.Client, input any) error {
	if p == nil {
		return nil
	}

	shouldInspect := false

	if p.NameRule != Allow || p.AddressRule != Allow || p.EmailRule != Allow || p.SsnRule != Allow || p.PasswordRule != Allow {
		shouldInspect = true
	}

	for _, cr := range p.CustomRules {
		if cr.Action != Allow {
			shouldInspect = true
		}
	}

	for _, regexr := range p.RegularExpressionRules {
		if regexr.Action != Allow {
			shouldInspect = true
		}
	}

	if !shouldInspect {
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
				return err
			}

			converted.Input = updated

		} else if input, ok := converted.Input.(string); ok {
			updated, err := p.inspect(client, []string{input})

			if err != nil {
				return err
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
			return err
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

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://localhost:8080/inspect", io.NopCloser(bytes.NewReader(data)))
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "application/json")

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	parsed := &Response{}
	err = json.Unmarshal(body, &parsed)
	if err != nil {
		return nil, err
	}

	blockedReasons := []string{}
	for blocked := range parsed.BlockedReasons {
		blockedReasons = append(blockedReasons, blocked)
	}

	warnings := []string{}
	for message := range parsed.Warnings {
		warnings = append(warnings, message)
	}

	if parsed.Action == Block {
		return nil, internal_errors.NewBlockedError(fmt.Sprintf("request blocked: %s", blockedReasons))
	}

	if len(parsed.Warnings) != 0 {
		return nil, internal_errors.NewWarningError(fmt.Sprintf("request warned: %s", warnings))
	}

	return parsed.Contents, nil
}
