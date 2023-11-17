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
}

func NewTokenCounter() *TokenCounter {
	tiktoken.SetBpeLoader(tiktoken_loader.NewOfflineLoader())
	return &TokenCounter{}
}

func (tc *TokenCounter) Count(model string, input string) (int, error) {
	encoder, err := tiktoken.GetEncoding("r50k_base")
	if err != nil {
		return 0, err
	}

	if strings.HasPrefix(model, "text-search") || strings.HasPrefix(model, "text-similarity") {
		token := encoder.Encode(input, nil, nil)
		return len(token), nil
	}

	encoder, err = tiktoken.EncodingForModel(model)
	if err != nil {
		return 0, err
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
