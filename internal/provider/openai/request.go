package openai

import "errors"

type ChatCompletionRequest struct {
	Model     string           `json:"model"`
	Messages  []RequestMessage `json:"messages"`
	Functions []Function       `json:"functions,omitempty"`
}

type ChatCompletionErrorContent struct {
	Message string `json:"message"`
	Type    string `json:"type"`
}

type FunctionCall struct {
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}

type RequestMessage struct {
	Name         string        `json:"name,omitempty"`
	FunctionCall *FunctionCall `json:"function_call,omitempty"`
	Role         string        `json:"role,omitempty"`
	Content      string        `json:"content,omitempty"`
}

type Function struct {
	Name        string           `json:"name,omitempty"`
	Description string           `json:"description,omitempty"`
	Parameters  *FuntionCallProp `json:"parameters,omitempty"`
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
