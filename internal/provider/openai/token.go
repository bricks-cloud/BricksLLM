package openai

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pkoukk/tiktoken-go"
	tiktoken_loader "github.com/pkoukk/tiktoken-go-loader"
)

type encoder interface {
	Encode(text string, allowedSpecial []string, disallowedSpecial []string) []int
}

type TokenCounter struct {
	encoderMap map[string]encoder
}

func NewTokenCounter() (*TokenCounter, error) {
	tiktoken.SetBpeLoader(tiktoken_loader.NewOfflineLoader())
	e, err := tiktoken.GetEncoding("cl100k_base")
	if err != nil {
		return nil, err
	}

	return &TokenCounter{
		encoderMap: map[string]encoder{
			"gpt-4":                  e,
			"gpt-4-0314":             e,
			"gpt-4-0613":             e,
			"gpt-4-32k":              e,
			"gpt-4-32k-0613":         e,
			"gpt-4-32k-0314":         e,
			"gpt-3.5-turbo":          e,
			"gpt-3.5-turbo-0301":     e,
			"gpt-3.5-turbo-0613":     e,
			"gpt-3.5-turbo-instruct": e,
			"gpt-3.5-turbo-16k":      e,
			"gpt-3.5-turbo-16k-0613": e,
		},
	}, nil
}

func (tc *TokenCounter) Count(model string, input string) (int, error) {
	encoder, ok := tc.encoderMap[model]
	if !ok {
		encoder, err := tiktoken.EncodingForModel(model)
		if err != nil {
			return 0, err
		}

		tc.encoderMap[model] = encoder
	}

	token := encoder.Encode(input, nil, nil)
	return len(token), nil
}

type functionCallProp interface {
	GetDescription() string
	GetType() string
	GetEnum() []string
	GetItems() (functionCallProp, error)
	GetRequired() []string
	GetProperties() (map[string]functionCallProp, error)
}

func formatObjectProperties(p functionCallProp, indent int) string {
	if p == nil {
		return ""
	}

	lines := []string{}
	properties, err := p.GetProperties()
	if err != nil {
		return ""
	}

	for name, param := range properties {
		description := param.GetDescription()
		if len(description) != 0 {
			lines = append(lines, fmt.Sprintf(`// %s`, description))
		}

		fieldRequired := false
		for _, requiredField := range param.GetRequired() {
			if requiredField == name {
				fieldRequired = true
			}
		}

		if fieldRequired {
			lines = append(lines, fmt.Sprintf(`%s: %s,`, name, formatType(param, indent)))
		}

		if !fieldRequired {
			lines = append(lines, fmt.Sprintf(`%s?: %s,`, name, formatType(param, indent)))
		}
	}

	formatedLines := []string{}
	for _, line := range lines {
		formatedLines = append(formatedLines, newSpaces(indent)+line)
	}

	return strings.Join(formatedLines, "\n")
}

func formatType(p functionCallProp, indent int) string {
	if p == nil {
		return "any"
	}

	switch p.GetType() {
	case "string":
		enums := p.GetEnum()
		if len(enums) != 0 {
			return fmt.Sprintf(`%s[]`, strings.Join(enums, " | "))
		}
		return "string"
	case "number":
		enums := p.GetEnum()
		if len(enums) != 0 {
			return fmt.Sprintf(`%s[]`, strings.Join(enums, " | "))
		}
		return "number"
	case "integer":
		enums := p.GetEnum()
		if len(enums) != 0 {
			return fmt.Sprintf(`%s[]`, strings.Join(enums, " | "))
		}
		return "integer"
	case "boolean":
		return "boolean"
	case "null":
		return "null"
	case "object":
		return strings.Join([]string{
			"{",
			formatObjectProperties(p, indent+2),
			"}",
		}, "\n")
	case "array":
		items, err := p.GetItems()
		if err != nil {
			return fmt.Sprintf(`%s[]`, formatType(items, indent))
		}

		return "any[]"
	}

	return "any"
}

func unmarshalInterface(source interface{}, dst interface{}) error {
	data, err := json.Marshal(source)
	if err != nil {
		return err
	}

	err = json.Unmarshal(data, dst)
	return err
}

func newSpaces(indent int) string {
	result := ""

	for i := 0; i < indent; i++ {
		result += " "
	}

	return result
}
