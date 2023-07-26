package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/bricks-cloud/atlas/internal/util"
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

type Provider string

const (
	openaiProvider Provider = "openai"
)

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
	system   OpenAiMessageRole = "system"
	user     OpenAiMessageRole = "user"
	assitant OpenAiMessageRole = "assitant"
	function OpenAiMessageRole = "function"
)

type OpenAiPrompt struct {
	Role    string `yaml:"role"`
	Content string `yaml:"content"`
}

type OpenAiModel string

const (
	gpt35Turbo OpenAiModel = "gpt-3.5-turbo"
)

type OpenAiRouteConfig struct {
	ApiCredential string          `yaml:"api_credential"`
	Model         OpenAiModel     `yaml:"model"`
	Prompts       []*OpenAiPrompt `yaml:"prompts"`
}

type OpenAiConfig struct {
	ApiCredential string `yaml:"api_credential"`
}

type Config struct {
	Routes       []*RouteConfig `yaml:"routes"`
	Server       *ServerConfig  `yaml:"server"`
	OpenAiConfig *OpenAiConfig  `yaml:"openai"`
}

func NewConfig(filePath string) (*Config, error) {
	yamlFile, err := ioutil.ReadFile(filePath)
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

	if len(c.Routes) == 0 {
		return nil, fmt.Errorf("routes are not configured in config file %s", filePath)
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

	if rc.Provider == openaiProvider {
		if rc.OpenAiConfig == nil {
			return errors.New("openai config is not provided")
		}

		for _, prompt := range rc.OpenAiConfig.Prompts {
			if len(prompt.Role) == 0 {
				return errors.New("role is not provided in openai prompt")
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
				return errors.New("referenced value in prompt does not exist")
			}

			if index != len(parts)-1 && value.DataType != ObjectDataType {
				return errors.New("input value is not represented as object")
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
