package vllm

import (
	"errors"

	goopenai "github.com/sashabaranov/go-openai"
)

type CompletionRequest struct {
	goopenai.CompletionRequest
	UseBeamSearch              bool    `json:"use_beam_search,omitempty"`
	TopK                       int     `json:"top_k,omitempty"`
	MinP                       int     `json:"min_p,omitempty"`
	RepetitionPenalty          float64 `json:"repetition_penalty,omitempty"`
	LengthPenalty              float64 `json:"length_penalty,omitempty"`
	EarlyStopping              bool    `json:"early_stopping,omitempty"`
	StopTokenIds               []int   `json:"stop_token_ids,omitempty"`
	IgnoreEos                  bool    `json:"ignore_eos,omitempty"`
	MinTokens                  int     `json:"min_tokens,omitempty"`
	SkipSpecialTokens          bool    `json:"skip_special_tokens,omitempty"`
	SpacesBetweenSpecialTokens bool    `json:"spaces_between_special_tokens,omitempty"`
}

type ChatRequest struct {
	goopenai.ChatCompletionRequest
	BestOf                     int     `json:"best_of,omitempty"`
	UseBeamSearch              bool    `json:"use_beam_search,omitempty"`
	TopK                       int     `json:"top_k,omitempty"`
	MinP                       int     `json:"min_p,omitempty"`
	RepetitionPenalty          float64 `json:"repetition_penalty,omitempty"`
	LengthPenalty              float64 `json:"length_penalty,omitempty"`
	EarlyStopping              bool    `json:"early_stopping,omitempty"`
	IgnoreEos                  bool    `json:"ignore_eos,omitempty"`
	MinTokens                  int     `json:"min_tokens,omitempty"`
	StopTokenIds               []int   `json:"stop_token_ids,omitempty"`
	SkipSpecialTokens          bool    `json:"skip_special_tokens,omitempty"`
	SpacesBetweenSpecialTokens bool    `json:"spaces_between_special_tokens,omitempty"`
}

type FuntionCallProp struct {
	Description string                 `json:"description,omitempty"`
	PropType    string                 `json:"type,omitempty"`
	Enum        []string               `json:"enum,omitempty"`
	Items       interface{}            `json:"items,omitempty"`
	Required    []string               `json:"required,omitempty"`
	Properties  map[string]interface{} `json:"properties,omitempty"`
}

func (p *FuntionCallProp) GetDescription() string {
	if p == nil {
		return ""
	}

	return p.Description
}

func (p *FuntionCallProp) GetType() string {
	if p == nil {
		return ""
	}

	return p.Description
}

func (p *FuntionCallProp) GetEnum() []string {
	if p == nil {
		return []string{}
	}

	return p.Enum
}

func (p *FuntionCallProp) GetItems() (functionCallProp, error) {
	if p == nil {
		return nil, errors.New("prop is nil")
	}

	items := &FuntionCallProp{}
	err := unmarshalInterface(p.Items, items)
	if err != nil {
		return nil, err
	}

	return items, nil
}

func (p *FuntionCallProp) GetRequired() []string {
	if p == nil {
		return []string{}
	}

	return p.Required
}

func (p *FuntionCallProp) GetProperties() (map[string]functionCallProp, error) {
	if p == nil {
		return nil, errors.New("prop is nil")
	}

	properties := map[string]functionCallProp{}
	for k, v := range p.Properties {
		property := &FuntionCallProp{}
		err := unmarshalInterface(v, property)
		if err != nil {
			return nil, err
		}

		properties[k] = property
	}

	return properties, nil
}
