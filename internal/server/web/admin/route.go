package admin

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/bricks-cloud/bricksllm/internal/route"
	"github.com/bricks-cloud/bricksllm/internal/stats"
	"github.com/bricks-cloud/bricksllm/internal/util"
	"github.com/gin-gonic/gin"
)

type RouteManager interface {
	GetRoute(id string) (*route.Route, error)
	GetRoutes() ([]*route.Route, error)
	CreateRoute(r *route.Route) (*route.Route, error)
}

func getCreateRouteHandler(m RouteManager, prod bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		log := util.GetLogFromCtx(c)
		stats.Incr("bricksllm.admin.get_create_route_handler.requests", nil, 1)

		start := time.Now()
		defer func() {
			dur := time.Since(start)
			stats.Timing("bricksllm.admin.get_create_route_handler.latency", dur, nil, 1)
		}()

		path := "/api/routes"
		if c == nil || c.Request == nil {
			c.JSON(http.StatusInternalServerError, &ErrorResponse{
				Type:     "/errors/empty-context",
				Title:    "context is empty error",
				Status:   http.StatusInternalServerError,
				Detail:   "gin context is empty",
				Instance: path,
			})
			return
		}

		data, err := io.ReadAll(c.Request.Body)
		if err != nil {
			logError(log, "error when reading create a route request body", prod, err)
			c.JSON(http.StatusInternalServerError, &ErrorResponse{
				Type:     "/errors/request-body-read",
				Title:    "request body reader error",
				Status:   http.StatusInternalServerError,
				Detail:   err.Error(),
				Instance: path,
			})
			return
		}

		r := &route.Route{}
		err = json.Unmarshal(data, r)
		if err != nil {
			logError(log, "error when unmarshalling create a route request body", prod, err)
			c.JSON(http.StatusInternalServerError, &ErrorResponse{
				Type:     "/errors/json-unmarshal",
				Title:    "json unmarshaller error",
				Status:   http.StatusInternalServerError,
				Detail:   err.Error(),
				Instance: path,
			})
			return
		}

		created, err := m.CreateRoute(r)
		if err != nil {
			errType := "internal"

			defer func() {
				stats.Incr("bricksllm.admin.get_create_route.create_route_error", []string{
					"error_type:" + errType,
				}, 1)
			}()

			if _, ok := err.(validationError); ok {
				errType = "validation"
				c.JSON(http.StatusBadRequest, &ErrorResponse{
					Type:     "/errors/validation",
					Title:    "custom provider validation failed",
					Status:   http.StatusBadRequest,
					Detail:   err.Error(),
					Instance: path,
				})
				return
			}

			logError(log, "error when creating a route", prod, err)
			c.JSON(http.StatusInternalServerError, &ErrorResponse{
				Type:     "/errors/route-manager",
				Title:    "creating a route error",
				Status:   http.StatusInternalServerError,
				Detail:   err.Error(),
				Instance: path,
			})
			return
		}

		stats.Incr("bricksllm.admin.get_create_route_handler.success", nil, 1)
		c.JSON(http.StatusOK, created)
	}
}

func getGetRouteHandler(m RouteManager, prod bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		log := util.GetLogFromCtx(c)
		stats.Incr("bricksllm.admin.get_get_route_handler.requests", nil, 1)

		start := time.Now()
		defer func() {
			dur := time.Since(start)
			stats.Timing("bricksllm.admin.get_get_route_handler.latency", dur, nil, 1)
		}()

		path := "/api/routes/:id"
		if c == nil || c.Request == nil {
			c.JSON(http.StatusInternalServerError, &ErrorResponse{
				Type:     "/errors/empty-context",
				Title:    "context is empty error",
				Status:   http.StatusInternalServerError,
				Detail:   "gin context is empty",
				Instance: path,
			})
			return
		}

		r, err := m.GetRoute(c.Param("id"))
		if err != nil {
			errType := "internal"
			defer func() {
				stats.Incr("bricksllm.admin.get_get_route_handler.get_custom_providers_err", []string{
					"error_type:" + errType,
				}, 1)
			}()

			if _, ok := err.(notFoundError); ok {
				errType = "not_found"

				logError(log, "route not found", prod, err)
				c.JSON(http.StatusNotFound, &ErrorResponse{
					Type:     "/errors/route-not-found",
					Title:    "route not found error",
					Status:   http.StatusNotFound,
					Detail:   err.Error(),
					Instance: path,
				})
				return
			}

			logError(log, "error when getting a route", prod, err)
			c.JSON(http.StatusInternalServerError, &ErrorResponse{
				Type:     "/errors/route-manager",
				Title:    "getting a route error",
				Status:   http.StatusInternalServerError,
				Detail:   err.Error(),
				Instance: path,
			})
			return
		}

		stats.Incr("bricksllm.admin.get_get_route_handler.success", nil, 1)
		c.JSON(http.StatusOK, r)
	}
}

func getGetRoutesHandler(m RouteManager, prod bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		log := util.GetLogFromCtx(c)
		stats.Incr("bricksllm.admin.get_get_routes_handler.requests", nil, 1)

		start := time.Now()
		defer func() {
			dur := time.Since(start)
			stats.Timing("bricksllm.admin.get_get_routes_handler.latency", dur, nil, 1)
		}()

		path := "/api/routes"
		if c == nil || c.Request == nil {
			c.JSON(http.StatusInternalServerError, &ErrorResponse{
				Type:     "/errors/empty-context",
				Title:    "context is empty error",
				Status:   http.StatusInternalServerError,
				Detail:   "gin context is empty",
				Instance: path,
			})
			return
		}

		rs, err := m.GetRoutes()
		if err != nil {
			errType := "internal"
			defer func() {
				stats.Incr("bricksllm.admin.get_get_routes_handler.get_custom_providers_err", []string{
					"error_type:" + errType,
				}, 1)
			}()

			logError(log, "error when getting a route", prod, err)
			c.JSON(http.StatusInternalServerError, &ErrorResponse{
				Type:     "/errors/route-manager",
				Title:    "getting a route error",
				Status:   http.StatusInternalServerError,
				Detail:   err.Error(),
				Instance: path,
			})
			return
		}

		stats.Incr("bricksllm.admin.get_get_routes_handler.success", nil, 1)
		c.JSON(http.StatusOK, rs)
	}
}
