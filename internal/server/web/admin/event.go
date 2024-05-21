package admin

import (
	"net/http"
	"time"

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
