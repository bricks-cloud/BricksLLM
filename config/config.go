package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/bricks-cloud/bricksllm/internal/util"
	"gopkg.in/yaml.v3"
)

type RateLimitConfig struct {
	Count    string `yaml:"count"`
	Interval string `yaml:"interval"`
}

type Protocol string

const (
	Http  Protocol = "http"
	Https Protocol = "https"
)

func (p Protocol) Valid() bool {
	if p != Http && p != Https {
		return false
	}

	return true
}

type Provider string

const (
	OpenaiProvider Provider = "openai"
)

func (p Provider) Valid() bool {
	if p != OpenaiProvider {
		return false
	}

	return true
}

type ApiLoggerConfig struct {
	HideIp      bool `yaml:"hide_ip"`
	HideHeaders bool `yaml:"hide_headers"`
}

func (alc *ApiLoggerConfig) GetHideIp() bool {
	if alc == nil {
		return false
	}

	return alc.HideIp
}

func (alc *ApiLoggerConfig) GetHideHeaders() bool {
	if alc == nil {
		return false
	}

	return alc.HideHeaders
}

type LlmLoggerConfig struct {
	HideHeaders         bool `yaml:"hide_headers"`
	HideResponseContent bool `yaml:"hide_response_content"`
	HidePromptContent   bool `yaml:"hide_prompt_content"`
}

func (llc *LlmLoggerConfig) GetHideResponseContent() bool {
	if llc == nil {
		return false
	}

	return llc.HideResponseContent
}

func (llc *LlmLoggerConfig) GetHidePromptContent() bool {
	if llc == nil {
		return false
	}

	return llc.HidePromptContent
}

func (llc *LlmLoggerConfig) GetHideHeaders() bool {
	if llc == nil {
		return false
	}

	return llc.HideHeaders
}

type LoggerConfig struct {
	Api *ApiLoggerConfig `yaml:"api"`
	Llm *LlmLoggerConfig `yaml:"llm"`
}

type ServerConfig struct {
	Port int `yaml:"port"`
}

type DataType string

const (
	StringDataType  DataType = "string"
	NumberDataType  DataType = "number"
	ArrayDataType   DataType = "array"
	ObjectDataType  DataType = "object"
	BooleanDataType DataType = "boolean"
)

func (d DataType) Valid() bool {
	if d != StringDataType && d != NumberDataType && d != ArrayDataType && d != ObjectDataType && d != BooleanDataType {
		return false
	}

	return true
}

type InputValue struct {
	DataType   DataType               `yaml:"type"`
	Properties map[string]interface{} `yaml:"properties"`
	Items      interface{}            `yaml:"items"`
}

type CorsConfig struct {
	AllowedOrgins      []string `yaml:"allowed_origins"`
	AllowedCredentials bool     `yaml:"allowed_credentials"`
}

func (cc *CorsConfig) Enabled() bool {
	return cc != nil
}

func (cc *CorsConfig) GetAllowedOrigins() []string {
	return cc.AllowedOrgins
}

func (cc *CorsConfig) GetAllowedCredentials() bool {
	return cc.AllowedCredentials
}

type KeyAuthConfig struct {
	Key string `yaml:"key"`
}

func (kac *KeyAuthConfig) Enabled() bool {
	return kac != nil
}

func (kac *KeyAuthConfig) GetKey() string {
	return kac.Key
}

type RouteConfig struct {
	Path                string                `yaml:"path"`
	CorsConfig          *CorsConfig           `yaml:"cors"`
	Input               map[string]InputValue `yaml:"input"`
	Provider            Provider              `yaml:"provider"`
	OpenAiConfig        *OpenAiRouteConfig    `yaml:"openai_config"`
	Description         string                `yaml:"description"`
	Protocol            Protocol              `yaml:"protocol"`
	CertFile            string                `yaml:"cert_file"`
	KeyFile             string                `yaml:"key_file"`
	KeyAuthConfig       *KeyAuthConfig        `yaml:"key_auth"`
	UpstreamSendTimeout time.Duration         `yaml:"upstream_send_time"`
}

type OpenAiMessageRole string

const (
	SystemMessageRole   OpenAiMessageRole = "system"
	UserMessageRole     OpenAiMessageRole = "user"
	AssitantMessageRole OpenAiMessageRole = "assistant"
	FunctionMessageRole OpenAiMessageRole = "function"
)

func (r OpenAiMessageRole) Valid() bool {
	if r != SystemMessageRole && r != UserMessageRole && r != AssitantMessageRole && r != FunctionMessageRole {
		return false
	}

	return true
}

type OpenAiPrompt struct {
	Role    OpenAiMessageRole `yaml:"role"`
	Content string            `yaml:"content"`
}

type OpenAiModel string

const (
	Gpt35Turbo        OpenAiModel = "gpt-3.5-turbo"
	Gpt35Turbo16k     OpenAiModel = "gpt-3.5-turbo-16k"
	Gpt35Turbo0613    OpenAiModel = "gpt-3.5-turbo-0613"
	Gpt35Turbo16k0613 OpenAiModel = "gpt-3.5-turbo-16k-0613"
	Gpt4              OpenAiModel = "gpt-4"
	Gpt40613          OpenAiModel = "gpt-4-0613"
	Gpt432k           OpenAiModel = "gpt-4-32k"
	Gpt432k0613       OpenAiModel = "gpt-4-32k-0613"
)

func (m OpenAiModel) Valid() bool {
	if m != Gpt35Turbo && m != Gpt35Turbo16k && m != Gpt35Turbo0613 && m != Gpt35Turbo16k0613 && m != Gpt4 && m != Gpt40613 && m != Gpt432k && m != Gpt432k0613 {
		return false
	}

	return true
}

type OpenAiRouteConfig struct {
	ApiCredential string          `yaml:"api_credential"`
	Model         OpenAiModel     `yaml:"model"`
	Prompts       []*OpenAiPrompt `yaml:"prompts"`
}

type OpenAiConfig struct {
	ApiCredential string `yaml:"api_credential"`
}

type Config struct {
	LoggerConfig *LoggerConfig  `yaml:"logger"`
	Routes       []*RouteConfig `yaml:"routes"`
	Server       *ServerConfig  `yaml:"server"`
	OpenAiConfig *OpenAiConfig  `yaml:"openai"`
}

func NewConfig(filePath string) (*Config, error) {
	yamlFile, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("unable to read config file with path %s: %w", filePath, err)
	}

	yamlFile = []byte(os.ExpandEnv(string(yamlFile)))
	c := &Config{}
	err = yaml.Unmarshal(yamlFile, c)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal yaml file with path %s: %w", filePath, err)
	}

	// default server port to 8080
	if c.Server == nil {
		c.Server = &ServerConfig{
			Port: 8080,
		}
	}

	// routes have to be configured
	if len(c.Routes) == 0 {
		return nil, fmt.Errorf("routes are not configured in config file %s", filePath)
	}

	// default server port to 8080
	if c.LoggerConfig == nil {
		c.LoggerConfig = &LoggerConfig{
			Api: &ApiLoggerConfig{
				HideIp:      false,
				HideHeaders: false,
			},

			Llm: &LlmLoggerConfig{
				HideHeaders:         false,
				HideResponseContent: false,
				HidePromptContent:   false,
			},
		}
	}

	apiCredentialConfigured := false
	if c.OpenAiConfig != nil && len(c.OpenAiConfig.ApiCredential) != 0 {
		apiCredentialConfigured = true
	}

	for _, route := range c.Routes {
		err = parseRouteConfig(route, apiCredentialConfigured)
		if err != nil {
			return nil, err
		}
	}
	return c, nil
}

func parseRouteConfig(rc *RouteConfig, isOpenAiConfigured bool) error {
	if len(rc.Path) == 0 {
		return errors.New("path is empty")
	}

	if len(rc.Provider) == 0 {
		return errors.New("provider is empty")
	}

	if !rc.Provider.Valid() {
		return errors.New("provider must be openai")
	}

	if rc.CorsConfig != nil {
		if len(rc.CorsConfig.AllowedOrgins) == 0 {
			return fmt.Errorf("cors config is present but allowed_origins is not specified for route: %s", rc.Path)
		}
	}

	if rc.KeyAuthConfig != nil {
		if len(rc.KeyAuthConfig.Key) == 0 {
			return fmt.Errorf("key_auth config is present but key is not specified for route: %s", rc.Path)
		}
	}

	if rc.Provider == OpenaiProvider {
		if rc.OpenAiConfig == nil {
			return errors.New("openai config is not provided")
		}

		if len(rc.OpenAiConfig.Model) == 0 {
			return errors.New("openai model cannot be empty")
		}

		if !rc.OpenAiConfig.Model.Valid() {
			return errors.New("open ai model must be of gpt-3.5-turbo, gpt-3.5-turbo-16k, gpt-3.5-turbo-0613, gpt-3.5-turbo-16k-0613, gpt-4, gpt-4-0613, gpt-4-32k and gpt-4-32k-0613")
		}

		for _, prompt := range rc.OpenAiConfig.Prompts {
			if len(prompt.Role) == 0 {
				return errors.New("role cannot be empty in openai prompt")
			}

			if !prompt.Role.Valid() {
				return errors.New("role must be of user, system, assistant or function")
			}

			if !isOpenAiConfigured && len(rc.OpenAiConfig.ApiCredential) == 0 {
				return errors.New("openai api credential is not configrued")
			}

			if len(prompt.Content) == 0 {
				return errors.New("content is not provided in openai prompt")
			}

			variableMap := util.GetVariableMap(prompt.Content)
			err := validateInput(rc.Input, variableMap)
			if err != nil {
				return err
			}
		}
	}

	if rc.Protocol == Https {
		if len(rc.CertFile) == 0 {
			return errors.New("cert file is not provided for https protocol")
		}

		if len(rc.KeyFile) == 0 {
			return errors.New("key file is not provided for https protocol")
		}
	}

	// defaut route protocol to http
	if len(rc.Protocol) == 0 {
		rc.Protocol = Http
	}

	if !rc.Protocol.Valid() {
		return errors.New("protocol must be of http or https")
	}

	return nil
}

func validateInput(input map[string]InputValue, variableMap map[string]string) error {
	if len(variableMap) == 0 {
		return nil
	}

	for _, reference := range variableMap {
		parts := strings.Split(reference, ".")

		if len(parts) == 0 {
			return errors.New("no references found inside `{{ }}` syntax")
		}

		if len(parts) == 1 {
			if _, found := input[parts[0]]; found {
				continue
			}

			return errors.New("referenced value in prompt is not defined in input")
		}

		innerInput := input
		for index, part := range parts {
			value, found := innerInput[part]
			if !found {
				return fmt.Errorf("referenced var: %s in prompt does not exist", value)
			}

			if index != len(parts)-1 && value.DataType != ObjectDataType {
				return fmt.Errorf("input value is not represented as object for referenced var: %s", reference)
			}

			if value.DataType == ObjectDataType && len(value.Properties) == 0 {
				return fmt.Errorf("object properties is empty for referenced var: %s", reference)
			}

			js, err := json.Marshal(value.Properties)
			if err != nil {
				return err
			}

			innerInput = map[string]InputValue{}
			err = json.Unmarshal(js, &innerInput)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
