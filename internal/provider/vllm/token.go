package vllm

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pkoukk/tiktoken-go"
	tiktoken_loader "github.com/pkoukk/tiktoken-go-loader"
)

type TokenCounter struct {
	encoder *tiktoken.Tiktoken
}

func NewTokenCounter() (*TokenCounter, error) {
	tiktoken.SetBpeLoader(tiktoken_loader.NewOfflineLoader())
	encoder, err := tiktoken.GetEncoding("r50k_base")
	if err != nil {
		return nil, err
	}

	return &TokenCounter{
		encoder: encoder,
	}, nil
}

func (tc *TokenCounter) Count(model string, input string) int {
	token := tc.encoder.Encode(input, nil, nil)
	return len(token)
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
