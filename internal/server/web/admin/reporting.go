package admin

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/bricks-cloud/bricksllm/internal/event"
	"github.com/bricks-cloud/bricksllm/internal/stats"
	"github.com/bricks-cloud/bricksllm/internal/util"
	"github.com/gin-gonic/gin"
)

func validateEventReportingRequest(r *event.ReportingRequest) bool {
	if r.Start == 0 || r.End == 0 || r.Increment <= 0 {
		return false
	}

	if r.Start >= r.End {
		return false
	}

	return true
}

func validateEventReportingByDayRequest(r *event.ReportingRequest) bool {
	if r.Start == 0 || r.End == 0 {
		return false
	}

	if r.Start >= r.End {
		return false
	}

	return true
}

func validateTopKeyReportingRequest(r *event.KeyReportingRequest) bool {
	if r.Start == 0 || r.End == 0 {
		return false
	}

	if r.Start >= r.End {
		return false
	}

	return true
}

func getGetEventMetricsHandler(m KeyReportingManager, prod bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		log := util.GetLogFromCtx(c)
		stats.Incr("bricksllm.admin.get_get_event_metrics.requests", nil, 1)

		start := time.Now()
		defer func() {
			dur := time.Since(start)
			stats.Timing("bricksllm.admin.get_get_event_metrics.latency", dur, nil, 1)
		}()

		path := "/api/reporting/events"

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
			logError(log, "error when reading event reporting request body", prod, err)
			c.JSON(http.StatusInternalServerError, &ErrorResponse{
				Type:     "/errors/request-body-read",
				Title:    "request body reader error",
				Status:   http.StatusInternalServerError,
				Detail:   err.Error(),
				Instance: path,
			})
			return
		}

		request := &event.ReportingRequest{}
		err = json.Unmarshal(data, request)
		if err != nil {
			logError(log, "error when unmarshalling event reporting request body", prod, err)
			c.JSON(http.StatusInternalServerError, &ErrorResponse{
				Type:     "/errors/json-unmarshal",
				Title:    "json unmarshaller error",
				Status:   http.StatusInternalServerError,
				Detail:   err.Error(),
				Instance: path,
			})
			return
		}

		if !validateEventReportingRequest(request) {
			stats.Incr("bricksllm.admin.get_get_event_metrics.request_not_valid", nil, 1)

			err = fmt.Errorf("event reporting request %+v is not valid", request)
			logError(log, "invalid reporting request", prod, err)
			c.JSON(http.StatusInternalServerError, &ErrorResponse{
				Type:     "/errors/invalid-reporting-request",
				Title:    "invalid reporting request",
				Status:   http.StatusBadRequest,
				Detail:   err.Error(),
				Instance: path,
			})
			return
		}

		reportingResponse, err := m.GetEventReporting(request)
		if err != nil {
			stats.Incr("bricksllm.admin.get_get_event_metrics.get_event_reporting_error", nil, 1)

			logError(log, "error when getting event reporting", prod, err)
			c.JSON(http.StatusInternalServerError, &ErrorResponse{
				Type:     "/errors/event-reporting-manager",
				Title:    "event reporting error",
				Status:   http.StatusInternalServerError,
				Detail:   err.Error(),
				Instance: path,
			})
			return
		}

		stats.Incr("bricksllm.admin.get_get_event_metrics.success", nil, 1)

		c.JSON(http.StatusOK, reportingResponse)
	}
}

func getGetEventMetricsByDayHandler(m KeyReportingManager, prod bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		log := util.GetLogFromCtx(c)
		stats.Incr("bricksllm.admin.get_get_event_metrics_by_day.requests", nil, 1)

		start := time.Now()
		defer func() {
			dur := time.Since(start)
			stats.Timing("bricksllm.admin.get_get_event_metrics_by_day.latency", dur, nil, 1)
		}()

		path := "/api/reporting/events-by-day"

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
			logError(log, "error when reading event by day reporting request body", prod, err)
			c.JSON(http.StatusInternalServerError, &ErrorResponse{
				Type:     "/errors/request-body-read",
				Title:    "request body reader error",
				Status:   http.StatusInternalServerError,
				Detail:   err.Error(),
				Instance: path,
			})
			return
		}

		request := &event.ReportingRequest{}
		err = json.Unmarshal(data, request)
		if err != nil {
			logError(log, "error when unmarshalling event by day reporting request body", prod, err)
			c.JSON(http.StatusInternalServerError, &ErrorResponse{
				Type:     "/errors/json-unmarshal",
				Title:    "json unmarshaller error",
				Status:   http.StatusInternalServerError,
				Detail:   err.Error(),
				Instance: path,
			})
			return
		}

		if !validateEventReportingByDayRequest(request) {
			stats.Incr("bricksllm.admin.get_get_event_metrics_by_day.request_not_valid", nil, 1)

			err = fmt.Errorf("event reporting request %+v is not valid", request)
			logError(log, "invalid reporting request", prod, err)
			c.JSON(http.StatusBadRequest, &ErrorResponse{
				Type:     "/errors/invalid-reporting-request",
				Title:    "invalid reporting request",
				Status:   http.StatusBadRequest,
				Detail:   err.Error(),
				Instance: path,
			})
			return
		}

		reportingResponse, err := m.GetAggregatedEventByDayReporting(request)
		if err != nil {
			stats.Incr("bricksllm.admin.get_get_event_metrics_by_day.get_aggregated_event_by_day_reporting", nil, 1)

			logError(log, "error when getting event by day reporting", prod, err)
			c.JSON(http.StatusInternalServerError, &ErrorResponse{
				Type:     "/errors/event-reporting-manager",
				Title:    "event reporting error",
				Status:   http.StatusInternalServerError,
				Detail:   err.Error(),
				Instance: path,
			})
			return
		}

		stats.Incr("bricksllm.admin.get_get_event_metrics_by_day.success", nil, 1)

		c.JSON(http.StatusOK, reportingResponse)
	}
}

func getGetTopKeysMetricsHandler(m KeyReportingManager, prod bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		log := util.GetLogFromCtx(c)
		stats.Incr("bricksllm.admin.get_get_top_keys_metrics_handler.requests", nil, 1)

		start := time.Now()
		defer func() {
			dur := time.Since(start)
			stats.Timing("bricksllm.admin.get_get_top_keys_metrics_handler.latency", dur, nil, 1)
		}()

		path := "/api/reporting/top-keys"

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
			logError(log, "error when reading top key reporting request body", prod, err)
			c.JSON(http.StatusInternalServerError, &ErrorResponse{
				Type:     "/errors/request-body-read",
				Title:    "request body reader error",
				Status:   http.StatusInternalServerError,
				Detail:   err.Error(),
				Instance: path,
			})
			return
		}

		request := &event.KeyReportingRequest{}
		err = json.Unmarshal(data, request)
		if err != nil {
			logError(log, "error when unmarshalling top key reporting request body", prod, err)
			c.JSON(http.StatusInternalServerError, &ErrorResponse{
				Type:     "/errors/json-unmarshal",
				Title:    "json unmarshaller error",
				Status:   http.StatusInternalServerError,
				Detail:   err.Error(),
				Instance: path,
			})
			return
		}

		if !validateTopKeyReportingRequest(request) {
			stats.Incr("bricksllm.admin.get_get_top_keys_metrics_handler.request_not_valid", nil, 1)
			err = fmt.Errorf("top key reporting request %+v is not valid", request)
			logError(log, "invalid reporting request", prod, err)
			c.JSON(http.StatusBadRequest, &ErrorResponse{
				Type:     "/errors/invalid-reporting-request",
				Title:    "invalid reporting request",
				Status:   http.StatusBadRequest,
				Detail:   err.Error(),
				Instance: path,
			})
			return
		}

		reportingResponse, err := m.GetTopKeyReporting(request)
		if err != nil {
			stats.Incr("bricksllm.admin.get_get_top_keys_metrics_handler.get_top_key_reporting", nil, 1)

			logError(log, "error when getting top key reporting", prod, err)
			c.JSON(http.StatusInternalServerError, &ErrorResponse{
				Type:     "/errors/event-reporting-manager",
				Title:    "top key reporting error",
				Status:   http.StatusInternalServerError,
				Detail:   err.Error(),
				Instance: path,
			})
			return
		}

		stats.Incr("bricksllm.admin.get_get_top_keys_metrics_handler.success", nil, 1)

		c.JSON(http.StatusOK, reportingResponse)
	}
}
