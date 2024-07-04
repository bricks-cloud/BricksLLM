package admin

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/bricks-cloud/bricksllm/internal/policy"
	"github.com/bricks-cloud/bricksllm/internal/telemetry"
	"github.com/bricks-cloud/bricksllm/internal/util"
	"github.com/gin-gonic/gin"
)

func getCreatePolicyHandler(pm PoliciesManager, prod bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		log := util.GetLogFromCtx(c)
		telemetry.Incr("bricksllm.admin.get_get_create_policy_handler.requests", nil, 1)

		start := time.Now()
		defer func() {
			dur := time.Since(start)
			telemetry.Timing("bricksllm.admin.get_get_create_policy_handler.latency", dur, nil, 1)
		}()

		path := "/api/policies"
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
			logError(log, "error when reading policy creation request body", prod, err)
			c.JSON(http.StatusInternalServerError, &ErrorResponse{
				Type:     "/errors/request-body-read",
				Title:    "request body reader error",
				Status:   http.StatusInternalServerError,
				Detail:   err.Error(),
				Instance: path,
			})
			return
		}

		p := &policy.Policy{}
		err = json.Unmarshal(data, p)
		if err != nil {
			logError(log, "error when unmarshalling policy creation request body", prod, err)
			c.JSON(http.StatusInternalServerError, &ErrorResponse{
				Type:     "/errors/json-unmarshal",
				Title:    "json unmarshaller error",
				Status:   http.StatusInternalServerError,
				Detail:   err.Error(),
				Instance: path,
			})
			return
		}

		created, err := pm.CreatePolicy(p)
		if err != nil {
			telemetry.Incr("bricksllm.admin.get_get_create_policy_handler.creat_policy_error", nil, 1)

			logError(log, "error when creating a policy", prod, err)
			c.JSON(http.StatusInternalServerError, &ErrorResponse{
				Type:     "/errors/policies/creation",
				Title:    "policy creation failed",
				Status:   http.StatusInternalServerError,
				Detail:   err.Error(),
				Instance: path,
			})
			return
		}

		telemetry.Incr("bricksllm.admin.get_get_create_policy_handler.success", nil, 1)

		c.JSON(http.StatusOK, created)
	}
}

func getUpdatePolicyHandler(pm PoliciesManager, prod bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		log := util.GetLogFromCtx(c)
		telemetry.Incr("bricksllm.admin.get_update_policy_handler.requests", nil, 1)

		start := time.Now()
		defer func() {
			dur := time.Since(start)
			telemetry.Timing("bricksllm.admin.get_update_policy_handler.latency", dur, nil, 1)
		}()

		path := "/api/policies"
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

		id := c.Param("id")
		if len(id) == 0 {
			c.JSON(http.StatusBadRequest, &ErrorResponse{
				Type:     "/errors/id-empty",
				Title:    "id is empty",
				Status:   http.StatusBadRequest,
				Detail:   "id is required for updating a policy.",
				Instance: path,
			})

			return
		}

		data, err := io.ReadAll(c.Request.Body)
		if err != nil {
			logError(log, "error when reading policy creation request body", prod, err)
			c.JSON(http.StatusInternalServerError, &ErrorResponse{
				Type:     "/errors/request-body-read",
				Title:    "request body reader error",
				Status:   http.StatusInternalServerError,
				Detail:   err.Error(),
				Instance: path,
			})
			return
		}

		p := &policy.UpdatePolicy{}
		err = json.Unmarshal(data, p)
		if err != nil {
			logError(log, "error when unmarshalling policy creation request body", prod, err)
			c.JSON(http.StatusInternalServerError, &ErrorResponse{
				Type:     "/errors/json-unmarshal",
				Title:    "json unmarshaller error",
				Status:   http.StatusInternalServerError,
				Detail:   err.Error(),
				Instance: path,
			})
			return
		}

		updated, err := pm.UpdatePolicy(id, p)
		if err != nil {
			telemetry.Incr("bricksllm.admin.get_update_policy_handler.update_policy_error", nil, 1)

			logError(log, "error when updating a policy by id", prod, err)
			c.JSON(http.StatusInternalServerError, &ErrorResponse{
				Type:     "/errors/policie/updates",
				Title:    "update a policy failed",
				Status:   http.StatusInternalServerError,
				Detail:   err.Error(),
				Instance: path,
			})
			return
		}

		telemetry.Incr("bricksllm.admin.get_update_policy_handler.success", nil, 1)

		c.JSON(http.StatusOK, updated)
	}
}

func getGetPoliciesByTagsHandler(pm PoliciesManager, prod bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		log := util.GetLogFromCtx(c)
		telemetry.Incr("bricksllm.admin.get_get_policies_by_tags_handler.requests", nil, 1)

		start := time.Now()
		defer func() {
			dur := time.Since(start)
			telemetry.Timing("bricksllm.admin.get_get_policies_by_tags_handler.latency", dur, nil, 1)
		}()

		path := "/api/policies"
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

		tags := c.QueryArray("tags")
		if len(tags) == 0 {
			c.JSON(http.StatusBadRequest, &ErrorResponse{
				Type:     "/errors/tags-empty",
				Title:    "query param tags is empty",
				Status:   http.StatusBadRequest,
				Detail:   "query param tags is required for retrieving policies.",
				Instance: path,
			})

			return
		}

		policies, err := pm.GetPoliciesByTags(c.QueryArray("tags"))
		if err != nil {
			errType := "internal"

			defer func() {
				telemetry.Incr("bricksllm.admin.get_get_policies_by_tags_handler.get_settings_error", []string{
					"error_type:" + errType,
				}, 1)
			}()

			logError(log, "error when getting policies by tags", prod, err)
			c.JSON(http.StatusInternalServerError, &ErrorResponse{
				Type:     "/errors/policies",
				Title:    "get policies by tags failed",
				Status:   http.StatusInternalServerError,
				Detail:   err.Error(),
				Instance: path,
			})
			return
		}

		telemetry.Incr("bricksllm.admin.get_get_policies_by_tags_handler.success", nil, 1)

		c.JSON(http.StatusOK, policies)
	}
}
