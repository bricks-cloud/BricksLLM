package manager

import (
	"errors"
	"fmt"
	"strings"
	"time"

	internal_errors "github.com/bricks-cloud/bricksllm/internal/errors"
	"github.com/bricks-cloud/bricksllm/internal/route"
	"github.com/bricks-cloud/bricksllm/internal/util"
)

type RoutesStorage interface {
	CreateRoute(r *route.Route) (*route.Route, error)
	GetRoute(id string) (*route.Route, error)
	GetRoutes() ([]*route.Route, error)
	GetRouteByPath(path string) (*route.Route, error)
}

type RoutesMemStorage interface {
	GetRoute(id string) *route.Route
}

type RouteManager struct {
	s  RoutesStorage
	ks Storage
	ms RoutesMemStorage
	ps ProviderSettingsMemStorage
}

func NewRouteManager(s RoutesStorage, ks Storage, ms RoutesMemStorage, psm ProviderSettingsMemStorage) *RouteManager {
	return &RouteManager{
		s:  s,
		ks: ks,
		ms: ms,
		ps: psm,
	}
}

func (m *RouteManager) GetRouteFromMemDb(path string) *route.Route {
	return m.ms.GetRoute(path)
}

func (m *RouteManager) GetRoute(id string) (*route.Route, error) {
	return m.s.GetRoute(id)
}

func (m *RouteManager) GetRoutes() ([]*route.Route, error) {
	return m.s.GetRoutes()
}

func (m *RouteManager) CreateRoute(r *route.Route) (*route.Route, error) {
	r.CreatedAt = time.Now().Unix()
	r.UpdatedAt = time.Now().Unix()
	r.Id = util.NewUuid()

	if err := m.validateRoute(r); err != nil {
		return nil, err
	}

	return m.s.CreateRoute(r)
}

func checkModelValidity(provider, model string) bool {
	if provider == "azure" {
		return contains(model, azureSupportedModels)
	}

	if provider == "openai" {
		return contains(model, openaiSupportedModels)
	}

	return false
}

var (
	azureSupportedModels = []string{
		"gpt-4-1106-preview",
		"gpt-4-1106-vision-preview",
		"gpt-4",
		"gpt-4-0314",
		"gpt-4-0613",
		"gpt-4-32k",
		"gpt-4-32k-0613",
		"gpt-4-32k-0314",
		"gpt-35-turbo",
		"gpt-35-turbo-1106",
		"gpt-35-turbo-0301",
		"gpt-35-turbo-instruct",
		"gpt-35-turbo-0613",
		"gpt-35-turbo-16k",
		"gpt-35-turbo-16k-0613",
		"ada",
	}

	openaiSupportedModels = []string{
		"gpt-4-1106-preview",
		"gpt-4-1106-vision-preview",
		"gpt-4",
		"gpt-4-0314",
		"gpt-4-0613",
		"gpt-4-32k",
		"gpt-4-32k-0613",
		"gpt-4-32k-0314",
		"gpt-3.5-turbo",
		"gpt-3.5-turbo-1106",
		"gpt-3.5-turbo-0301",
		"gpt-3.5-turbo-instruct",
		"gpt-3.5-turbo-0613",
		"gpt-3.5-turbo-16k",
		"gpt-3.5-turbo-16k-0613",
		"text-embedding-ada-002",
	}

	supportedModels = []string{
		"gpt-4-1106-preview",
		"gpt-4-1106-vision-preview",
		"gpt-4",
		"gpt-4-0314",
		"gpt-4-0613",
		"gpt-4-32k",
		"gpt-4-32k-0613",
		"gpt-4-32k-0314",
		"gpt-35-turbo",
		"gpt-35-turbo-1106",
		"gpt-35-turbo-0301",
		"gpt-35-turbo-instruct",
		"gpt-35-turbo-0613",
		"gpt-35-turbo-16k",
		"gpt-35-turbo-16k-0613",
		"gpt-3.5-turbo",
		"gpt-3.5-turbo-1106",
		"gpt-3.5-turbo-0301",
		"gpt-3.5-turbo-instruct",
		"gpt-3.5-turbo-0613",
		"gpt-3.5-turbo-16k",
		"gpt-3.5-turbo-16k-0613",
		"ada",
		"text-embedding-ada-002",
	}

	adaModels = []string{
		"ada",
		"text-embedding-ada-002",
	}

	chatCompletionModels = []string{
		"gpt-35-turbo",
		"gpt-35-turbo-1106",
		"gpt-35-turbo-0301",
		"gpt-35-turbo-instruct",
		"gpt-35-turbo-0613",
		"gpt-35-turbo-16k",
		"gpt-35-turbo-16k-0613",
		"gpt-4-1106-preview",
		"gpt-4-1106-vision-preview",
		"gpt-4",
		"gpt-4-0314",
		"gpt-4-0613",
		"gpt-4-32k",
		"gpt-4-32k-0613",
		"gpt-4-32k-0314",
		"gpt-3.5-turbo",
		"gpt-3.5-turbo-1106",
		"gpt-3.5-turbo-0301",
		"gpt-3.5-turbo-instruct",
		"gpt-3.5-turbo-0613",
		"gpt-3.5-turbo-16k",
		"gpt-3.5-turbo-16k-0613",
	}

	supportedProviders = []string{
		"openai",
		"azure",
	}
)

func contains(target string, source []string) bool {
	for _, s := range source {
		if s == target {
			return true
		}
	}

	return false
}

func (m *RouteManager) validateRoute(r *route.Route) error {
	fields := []string{}

	if len(r.Name) == 0 {
		fields = append(fields, "name")
	}

	if len(r.Path) == 0 {
		fields = append(fields, "path")
	}

	if len(r.KeyIds) == 0 {
		fields = append(fields, "keyIds")
	}

	if len(r.Steps) == 0 {
		fields = append(fields, "steps")
	}

	containAda := false

	for index, step := range r.Steps {
		if len(step.Provider) == 0 {
			fields = append(fields, fmt.Sprintf("steps.[%d].provider", index))
		}

		if !contains(step.Provider, supportedProviders) {
			return errors.New(fmt.Sprintf("steps.[%d].provider is not supported. Only azure and openai are supported", index))
		}

		if step.Provider == "azure" {
			apiVersion, _ := step.Params["apiVersion"]
			if len(apiVersion) == 0 {
				fields = append(fields, fmt.Sprintf("steps.[%d].params.apiVersion", index))
			}

			deploymentId, _ := step.Params["deploymentId"]
			if len(deploymentId) == 0 {
				fields = append(fields, fmt.Sprintf("steps.[%d].params.deploymentId", index))
			}
		}

		if len(step.Model) == 0 {
			fields = append(fields, fmt.Sprintf("steps.[%d].model", index))
		}

		if !contains(step.Model, supportedModels) {
			return errors.New(fmt.Sprintf("steps.[%d].model is not supported. Only chat completion and embeddings model are supported.", index))
		}

		if !checkModelValidity(step.Provider, step.Model) {
			return errors.New(fmt.Sprintf("model: %s is not supported for provider: %s.", step.Model, step.Provider))
		}

		if !containAda && contains(step.Model, adaModels) {
			containAda = true
		}
	}

	for _, step := range r.Steps {
		if containAda && !contains(step.Model, adaModels) {
			return errors.New("steps must have congruent models. Chat completion and embedding models cannot be in the same route config.")
		}

		if !containAda && !contains(step.Model, chatCompletionModels) {
			return errors.New("steps must have congruent models. Chat completion and embedding models cannot be in the same route config.")
		}
	}

	if r.CacheConfig == nil {
		fields = append(fields, "cacheConfig")
	}

	if r.CacheConfig != nil {
		parsed, err := time.ParseDuration(r.CacheConfig.Ttl)
		if err != nil {
			fields = append(fields, "cacheConfig.ttl")
		}

		max := time.Hour * 720

		if parsed > max {
			return internal_errors.NewValidationError("cacheConfig.ttl exceedes 30 days")
		}
	}

	found, err := m.ks.GetKeys(nil, r.KeyIds, "")
	if err != nil {
		return err
	}

	for _, key := range found {
		settingIds := key.GetSettingIds()
		settings := m.ps.GetSettings(settingIds)

		if !r.ValidateSettings(settings) {
			return errors.New("provider settings assosciated with the key cannot for accessing models specified in the route")
		}
	}

	_, err = m.s.GetRouteByPath(r.Path)
	if err == nil {
		return internal_errors.NewValidationError("path is not unique")
	}

	if _, ok := err.(notFoundError); !ok {
		return err
	}

	if len(found) != len(r.KeyIds) {
		return internal_errors.NewValidationError("specified key ids are not found")
	}

	if len(fields) != 0 {
		return internal_errors.NewValidationError(fmt.Sprintf("invalid fields in route: %s", strings.Join(fields, ",")))
	}

	return nil
}
