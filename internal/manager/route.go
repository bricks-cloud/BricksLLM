package manager

import (
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
}

func NewRouteManager(s RoutesStorage, ks Storage, ms RoutesMemStorage) *RouteManager {
	return &RouteManager{
		s:  s,
		ks: ks,
		ms: ms,
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

	for index, step := range r.Steps {
		if len(step.Provider) == 0 {
			fields = append(fields, fmt.Sprintf("steps.[%d].provider", index))
		}

		if step.Provider == "azure" && len(step.Params) == 0 {
			apiVersion, _ := step.Params["apiVersion"]
			if len(apiVersion) == 0 {
				fields = append(fields, fmt.Sprintf("steps.[%d].params.apiVersion", index))
			}

			deploymentId, _ := step.Params["deploymentId"]
			if len(deploymentId) == 0 {
				fields = append(fields, fmt.Sprintf("steps.[%d].params.deploymentId", index))
			}

			fields = append(fields, fmt.Sprintf("steps.[%d].provider", index))
		}

		if len(step.Model) == 0 {
			fields = append(fields, fmt.Sprintf("steps.[%d].model", index))
		}
	}

	if r.CacheConfig == nil {
		fields = append(fields, "cacheConfig")
	}

	if r.CacheConfig != nil {
		if r.CacheConfig.Enabled && len(r.CacheConfig.Ttl) == 0 {
			fields = append(fields, "cacheConfig.ttl")
		}
	}

	found, err := m.ks.GetKeys(nil, r.KeyIds, "")
	if err != nil {
		return err
	}

	if len(found) != len(r.KeyIds) {
		return internal_errors.NewValidationError("specified key ids are not found")
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
