package manager

import (
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"

	internal_errors "github.com/bricks-cloud/bricksllm/internal/errors"
	"github.com/bricks-cloud/bricksllm/internal/provider/custom"
	"github.com/bricks-cloud/bricksllm/internal/util"
)

type CustomProvidersStorage interface {
	CreateCustomProvider(provider *custom.Provider) (*custom.Provider, error)
	GetCustomProviders() ([]*custom.Provider, error)
	GetCustomProviderByName(name string) (*custom.Provider, error)
	GetCustomProvider(id string) (*custom.Provider, error)
	UpdateCustomProvider(id string, provider *custom.UpdateProvider) (*custom.Provider, error)
}

type CustomProvidersMemStorage interface {
	GetProvider(name string) *custom.Provider
	GetRouteConfig(name, path string) *custom.RouteConfig
}

type CustomProvidersManager struct {
	Storage CustomProvidersStorage
	Mem     CustomProvidersMemStorage
}

func NewCustomProvidersManager(s CustomProvidersStorage, mem CustomProvidersMemStorage) *CustomProvidersManager {
	return &CustomProvidersManager{
		Storage: s,
		Mem:     mem,
	}
}

func containsSpace(str string) bool {
	for _, c := range str {
		if unicode.IsSpace(c) {
			return true
		}
	}
	return false
}

func gatherEmptyFieldsFromRouteConfig(index int, rc *custom.RouteConfig) []string {
	invalidFields := []string{}

	if len(rc.Path) == 0 {
		invalidFields = append(invalidFields, fmt.Sprintf("route_configs.[%d].path", index))
	}

	if len(rc.TargetUrl) == 0 {
		invalidFields = append(invalidFields, fmt.Sprintf("route_configs.[%d].target_url", index))
	}

	if len(rc.ModelLocation) == 0 {
		invalidFields = append(invalidFields, fmt.Sprintf("route_configs.[%d].model_location", index))
	}

	if len(rc.RequestPromptLocation) == 0 {
		invalidFields = append(invalidFields, fmt.Sprintf("route_configs.[%d].request_prompt_location", index))
	}

	if len(rc.ResponseCompletionLocation) == 0 {
		invalidFields = append(invalidFields, fmt.Sprintf("route_configs.[%d].response_completion_location", index))
	}

	return invalidFields

}

func validateCustomProviderUpdate(existing *custom.Provider, updated *custom.UpdateProvider) error {
	invalidFields := []string{}
	pathToRouteMap := map[string]*custom.RouteConfig{}

	for _, rc := range existing.RouteConfigs {
		pathToRouteMap[rc.Path] = rc
	}

	for index, rc := range updated.RouteConfigs {
		_, ok := pathToRouteMap[rc.Path]

		if len(rc.StreamLocation) != 0 {
			if len(rc.StreamEndWord) == 0 {
				invalidFields = append(invalidFields, fmt.Sprintf("route_configs.[%d].stream_end_word", index))
			}

			if len(rc.StreamResponseCompletionLocation) == 0 {
				invalidFields = append(invalidFields, fmt.Sprintf("route_configs.[%d].stream_response_completion_location", index))
			}
		}

		if !ok {
			duplicates := map[string]struct{}{}
			_, ok := duplicates[rc.Path]
			if ok {
				return internal_errors.NewValidationError("route configs cannot contain duplicated paths")
			}

			if !ok {
				duplicates[rc.Path] = struct{}{}
			}

			invalidFields = append(invalidFields, gatherEmptyFieldsFromRouteConfig(index, rc)...)
		}
	}

	if len(invalidFields) != 0 {
		return internal_errors.NewValidationError(fmt.Sprintf("empty fields in provider: %s", strings.Join(invalidFields, ",")))
	}

	return nil
}

func validateCustomProviderCreation(provider *custom.Provider) error {
	invalidFields := []string{}

	if provider.Provider == "openai" || provider.Provider == "anthropic" || provider.Provider == "azure" || provider.Provider == "deepinfra" || provider.Provider == "vllm" {
		return internal_errors.NewValidationError("provider cannot be named openai or anthropic")
	}

	if len(provider.Provider) == 0 {
		invalidFields = append(invalidFields, "provider")
	}

	if containsSpace(provider.Provider) {
		return internal_errors.NewValidationError("provider cannot contain white space")
	}

	if len(provider.RouteConfigs) == 0 {
		invalidFields = append(invalidFields, "route_configs")
	}

	if len(provider.RouteConfigs) != 0 {
		for index, rc := range provider.RouteConfigs {
			duplicates := map[string]struct{}{}

			if containsSpace(rc.Path) {
				return internal_errors.NewValidationError("route configs path cannot contain white space")
			}

			if !strings.HasPrefix(rc.Path, "/") {
				return internal_errors.NewValidationError(`route configs path must start with "/"`)
			}

			_, ok := duplicates[rc.Path]
			if ok {
				return internal_errors.NewValidationError("route configs cannot contain duplicated paths")
			}

			if !ok {
				duplicates[rc.Path] = struct{}{}
			}

			if len(rc.StreamLocation) != 0 {
				if len(rc.StreamEndWord) == 0 {
					invalidFields = append(invalidFields, fmt.Sprintf("route_configs.[%d].stream_end_word", index))
				}

				if len(rc.StreamResponseCompletionLocation) == 0 {
					invalidFields = append(invalidFields, fmt.Sprintf("route_configs.[%d].stream_response_completion_location", index))
				}
			}

			invalidFields = append(invalidFields, gatherEmptyFieldsFromRouteConfig(index, rc)...)
		}
	}

	if len(provider.AuthenticationParam) != 0 && containsSpace(provider.AuthenticationParam) {
		return internal_errors.NewValidationError("authentication_param cannot contain white space")
	}

	if len(invalidFields) != 0 {
		return internal_errors.NewValidationError(fmt.Sprintf("empty fields in provider: %s", strings.Join(invalidFields, ",")))
	}

	return nil
}

type notFoundError interface {
	Error() string
	NotFound()
}

func (m *CustomProvidersManager) CreateCustomProvider(provider *custom.Provider) (*custom.Provider, error) {
	err := validateCustomProviderCreation(provider)
	if err != nil {
		return nil, err
	}

	name := strings.ToLower(provider.Provider)

	_, err = m.Storage.GetCustomProviderByName(name)

	if err == nil {
		return nil, errors.New("provider must be unique")
	}

	_, ok := err.(notFoundError)
	if !ok {
		return nil, err
	}

	provider.Id = util.NewUuid()
	provider.CreatedAt = time.Now().Unix()
	provider.UpdatedAt = time.Now().Unix()
	provider.Provider = strings.ToLower(provider.Provider)
	provider.AuthenticationParam = strings.ToLower(provider.AuthenticationParam)

	return m.Storage.CreateCustomProvider(provider)
}

func (m *CustomProvidersManager) GetCustomProviderFromMem(name string) *custom.Provider {
	return m.Mem.GetProvider(name)
}

func (m *CustomProvidersManager) GetRouteConfigFromMem(name, path string) *custom.RouteConfig {
	return m.Mem.GetRouteConfig(name, path)
}

func (m *CustomProvidersManager) GetCustomProviders() ([]*custom.Provider, error) {
	return m.Storage.GetCustomProviders()
}

func (m *CustomProvidersManager) UpdateCustomProvider(id string, provider *custom.UpdateProvider) (*custom.Provider, error) {
	provider.UpdatedAt = time.Now().Unix()

	existing, err := m.Storage.GetCustomProvider(id)
	if err != nil {
		return nil, err
	}

	err = validateCustomProviderUpdate(existing, provider)
	if err != nil {
		return nil, err
	}

	return m.Storage.UpdateCustomProvider(id, provider)
}
