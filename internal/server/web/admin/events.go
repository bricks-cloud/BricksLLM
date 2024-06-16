package admin

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/bricks-cloud/bricksllm/internal/event"
	"github.com/bricks-cloud/bricksllm/internal/stats"
	"github.com/bricks-cloud/bricksllm/internal/util"
	"github.com/gin-gonic/gin"
)

func getGetUserIdsHandler(m KeyReportingManager, prod bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		log := util.GetLogFromCtx(c)
		stats.Incr("bricksllm.admin.get_get_user_ids_handler.requests", nil, 1)

		start := time.Now()
		defer func() {
			dur := time.Since(start)
			stats.Timing("bricksllm.admin.get_get_user_ids_handler.latency", dur, nil, 1)
		}()

		path := "/api/reporting/user-ids"
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

		kid := c.Query("keyId")
		if len(kid) == 0 {
			c.JSON(http.StatusBadRequest, &ErrorResponse{
				Type:     "/errors/missing-key-id",
				Title:    "key id query param is missing",
				Status:   http.StatusBadRequest,
				Detail:   "key id query is missing",
				Instance: path,
			})
			return
		}

		cids, err := m.GetUserIds(kid)
		if err != nil {
			stats.Incr("bricksllm.admin.get_get_user_ids_handler.get_user_ids_err", nil, 1)

			logError(log, "error when getting userIds", prod, err)
			c.JSON(http.StatusInternalServerError, &ErrorResponse{
				Type:     "/errors/key-reporting-manager",
				Title:    "getting user ids error",
				Status:   http.StatusInternalServerError,
				Detail:   err.Error(),
				Instance: path,
			})
			return
		}

		stats.Incr("bricksllm.admin.get_get_user_ids_handler.success", nil, 1)
		c.JSON(http.StatusOK, cids)
	}
}

func getGetCustomIdsHandler(m KeyReportingManager, prod bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		log := util.GetLogFromCtx(c)
		stats.Incr("bricksllm.admin.get_get_custom_ids_handler.requests", nil, 1)

		start := time.Now()
		defer func() {
			dur := time.Since(start)
			stats.Timing("bricksllm.admin.get_get_custom_ids_handler.latency", dur, nil, 1)
		}()

		path := "/api/reporting/custom-ids"
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

		kid := c.Query("keyId")

		if len(kid) == 0 {
			c.JSON(http.StatusBadRequest, &ErrorResponse{
				Type:     "/errors/missing-key-id",
				Title:    "key id query param is missing",
				Status:   http.StatusBadRequest,
				Detail:   "key id query is missing",
				Instance: path,
			})
			return
		}

		cids, err := m.GetCustomIds(kid)
		if err != nil {
			stats.Incr("bricksllm.admin.get_get_user_ids_handler.get_custom_ids_err", nil, 1)

			logError(log, "error when getting custom ids", prod, err)
			c.JSON(http.StatusInternalServerError, &ErrorResponse{
				Type:     "/errors/key-reporting-manager",
				Title:    "getting custom ids error",
				Status:   http.StatusInternalServerError,
				Detail:   err.Error(),
				Instance: path,
			})
			return
		}

		stats.Incr("bricksllm.admin.get_get_custom_ids_handler.success", nil, 1)
		c.JSON(http.StatusOK, cids)
	}
}

func getGetEventsHandler(m KeyReportingManager, prod bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		log := util.GetLogFromCtx(c)
		stats.Incr("bricksllm.admin.get_get_events_handler.requests", nil, 1)

		start := time.Now()
		defer func() {
			dur := time.Since(start)
			stats.Timing("bricksllm.admin.get_get_events_handler.latency", dur, nil, 1)
		}()

		path := "/api/events"

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

		customId, ciok := c.GetQuery("customId")
		userId, uiok := c.GetQuery("userId")
		keyIds, kiok := c.GetQueryArray("keyIds")
		if !ciok && !kiok && !uiok {
			c.JSON(http.StatusBadRequest, &ErrorResponse{
				Type:     "/errors/no-filters-empty",
				Title:    "none of customId, keyIds and userId is specified",
				Status:   http.StatusBadRequest,
				Detail:   "customId, userId and keyIds are empty. one of them is required for retrieving events.",
				Instance: path,
			})

			return
		}

		var qstart int64 = 0
		var qend int64 = 0

		if kiok {
			startstr, sok := c.GetQuery("start")
			if !sok {
				c.JSON(http.StatusBadRequest, &ErrorResponse{
					Type:     "/errors/query-param-start-missing",
					Title:    "query param start is missing",
					Status:   http.StatusBadRequest,
					Detail:   "start query param is not provided",
					Instance: path,
				})

				return
			}

			parsedStart, err := strconv.ParseInt(startstr, 10, 64)
			if err != nil {
				c.JSON(http.StatusBadRequest, &ErrorResponse{
					Type:     "/errors/bad-start-query-param",
					Title:    "start query cannot be parsed",
					Status:   http.StatusBadRequest,
					Detail:   "start query param must be int64",
					Instance: path,
				})

				return
			}

			qstart = parsedStart

			endstr, eoi := c.GetQuery("end")
			if !eoi {
				c.JSON(http.StatusBadRequest, &ErrorResponse{
					Type:     "/errors/query-param-end-missing",
					Title:    "query param end is missing",
					Status:   http.StatusBadRequest,
					Detail:   "end query param is not provided",
					Instance: path,
				})

				return
			}

			parsedEnd, err := strconv.ParseInt(endstr, 10, 64)
			if err != nil {
				c.JSON(http.StatusBadRequest, &ErrorResponse{
					Type:     "/errors/bad-end-query-param",
					Title:    "end query cannot be parsed",
					Status:   http.StatusBadRequest,
					Detail:   "end query param must be int64",
					Instance: path,
				})

				return
			}

			qend = parsedEnd
		}

		evs, err := m.GetEvents(userId, customId, keyIds, qstart, qend)
		if err != nil {
			stats.Incr("bricksllm.admin.get_get_events_handler.get_events_error", nil, 1)

			logError(log, "error when getting events", prod, err)
			c.JSON(http.StatusInternalServerError, &ErrorResponse{
				Type:     "/errors/event-manager",
				Title:    "getting events error",
				Status:   http.StatusInternalServerError,
				Detail:   err.Error(),
				Instance: path,
			})
			return
		}

		stats.Incr("bricksllm.admin.get_get_events_handler.success", nil, 1)

		c.JSON(http.StatusOK, evs)
	}
}

func getGetEventsV2Handler(m KeyReportingManager, prod bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		log := util.GetLogFromCtx(c)
		stats.Incr("bricksllm.admin.get_get_events_v2_handler.requests", nil, 1)

		start := time.Now()
		defer func() {
			dur := time.Since(start)
			stats.Timing("bricksllm.admin.get_get_events_v2_handler.latency", dur, nil, 1)
		}()

		path := "/api/v2/events"
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
			logError(log, "error when reading get events request body", prod, err)
			c.JSON(http.StatusInternalServerError, &ErrorResponse{
				Type:     "/errors/request-body-read",
				Title:    "get events request body reader error",
				Status:   http.StatusInternalServerError,
				Detail:   err.Error(),
				Instance: path,
			})
			return
		}

		request := &event.EventRequest{}
		err = json.Unmarshal(data, request)
		if err != nil {
			logError(log, "error when unmarshalling get events request body", prod, err)
			c.JSON(http.StatusInternalServerError, &ErrorResponse{
				Type:     "/errors/json-unmarshal",
				Title:    "json unmarshaller error",
				Status:   http.StatusInternalServerError,
				Detail:   err.Error(),
				Instance: path,
			})
			return
		}

		keys, err := m.GetEventsV2(request)
		if err != nil {
			errType := "internal"

			defer func() {
				stats.Incr("bricksllm.admin.get_get_events_v2_handler.get_events_v2_err", []string{
					"error_type:" + errType,
				}, 1)
			}()

			if _, ok := err.(validationError); ok {
				errType = "validation"
				c.JSON(http.StatusBadRequest, &ErrorResponse{
					Type:     "/errors/validation",
					Title:    "get events request validation failed",
					Status:   http.StatusBadRequest,
					Detail:   err.Error(),
					Instance: path,
				})
				return
			}

			logError(log, "error when getting events", prod, err)
			c.JSON(http.StatusInternalServerError, &ErrorResponse{
				Type:     "/errors/event-manager",
				Title:    "getting events errored out",
				Status:   http.StatusInternalServerError,
				Detail:   err.Error(),
				Instance: path,
			})
			return
		}

		stats.Incr("bricksllm.admin.get_get_events_v2_handler.success", nil, 1)
		c.JSON(http.StatusOK, keys)
	}
}
