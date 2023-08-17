package web

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/bricks-cloud/bricksllm/internal/key"
	"github.com/bricks-cloud/bricksllm/internal/logger"
	"github.com/gin-gonic/gin"
)

type KeyManager interface {
	GetKeysByTag(tag string) ([]*key.ResponseKey, error)
	UpdateKey(id string, key *key.UpdateKey) (*key.ResponseKey, error)
	CreateKey(key *key.RequestKey) (*key.ResponseKey, error)
	DeleteKey(id string) error
}

type ErrorResponse struct {
	Type     string `json:"type"`
	Title    string `json:"title"`
	Status   int    `json:"status"`
	Detail   string `json:"detail"`
	Instance string `json:"instance"`
}

type AdminServer struct {
	server *http.Server
	logger logger.Logger
	m      KeyManager
}

func NewAdminServer(log logger.Logger, m KeyManager) (*AdminServer, error) {
	router := gin.New()

	router.GET("/api/key-management/keys", getGetKeysHandler(m))
	router.PUT("/api/key-management/keys", getCreateKeyHandler(m))
	router.PATCH("/api/key-management/keys/:id", getUpdateKeyHandler(m))
	router.DELETE("/api/key-management/keys/:id", getDeleteKeyHandler(m))

	srv := &http.Server{
		Addr:    ":8001",
		Handler: router,
	}

	log.Info("GET    /api/key-management/keys is set up for retrieving keys using a query param called tag")
	log.Info("PUT    /api/key-management/keys is set up for creating a key")
	log.Info("PATCH  /api/key-management/keys/:id is set up for updating a key using an id")
	log.Info("DELETE /api/key-management/keys/:id is set up for deleting a key using an id")

	return &AdminServer{
		logger: log,
		server: srv,
		m:      m,
	}, nil
}

func (as *AdminServer) Run() {
	go func() {
		if err := as.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			as.logger.Fatalf("error admin server listening: %v", err)
		}

		as.logger.Info("admin server listening at 8001")
	}()
}

func (as *AdminServer) Shutdown(ctx context.Context) error {
	if err := as.server.Shutdown(ctx); err != nil {
		as.logger.Debugf("error shutting down admin server: %v", err)

		return err
	}

	return nil
}

func getGetKeysHandler(m KeyManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		tag := c.Query("tag")
		path := "/api/key-management/keys"
		if len(tag) == 0 {
			c.JSON(http.StatusBadRequest, &ErrorResponse{
				Type:     "/errors/missing-query-tag",
				Title:    "tag is empty",
				Status:   http.StatusBadRequest,
				Detail:   "query param tag is missing from the request url. it is required for retrieving keys.",
				Instance: path,
			})
			return
		}

		keys, err := m.GetKeysByTag(tag)
		if err != nil {
			c.JSON(http.StatusBadRequest, &ErrorResponse{
				Type:     "/errors/getting-keys",
				Title:    "getting keys errored out",
				Status:   http.StatusInternalServerError,
				Detail:   err.Error(),
				Instance: path,
			})
			return
		}

		c.JSON(http.StatusOK, keys)
	}
}

type ValidationError interface {
	Error() string
	Validation()
}

func getCreateKeyHandler(m KeyManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		path := "/api/key-management/keys"
		data, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusInternalServerError, &ErrorResponse{
				Type:     "/errors/request-body-read",
				Title:    "request body reader error",
				Status:   http.StatusInternalServerError,
				Detail:   err.Error(),
				Instance: path,
			})
			return
		}

		rk := &key.RequestKey{}
		err = json.Unmarshal(data, rk)
		if err != nil {
			c.JSON(http.StatusInternalServerError, &ErrorResponse{
				Type:     "/errors/json-unmarshal",
				Title:    "json unmarshaller error",
				Status:   http.StatusInternalServerError,
				Detail:   err.Error(),
				Instance: path,
			})
			return
		}

		resk, err := m.CreateKey(rk)
		if err != nil {
			if _, ok := err.(ValidationError); ok {
				c.JSON(http.StatusBadRequest, &ErrorResponse{
					Type:     "/errors/validation",
					Title:    "key validation failed",
					Status:   http.StatusBadRequest,
					Detail:   err.Error(),
					Instance: path,
				})
				return
			}

			c.JSON(http.StatusInternalServerError, &ErrorResponse{
				Type:     "/errors/key-manager",
				Title:    "key creation error",
				Status:   http.StatusInternalServerError,
				Detail:   err.Error(),
				Instance: path,
			})
			return
		}

		c.JSON(http.StatusOK, resk)
	}
}

func getUpdateKeyHandler(m KeyManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		path := "/api/key-management/keys/:id"
		if len(id) == 0 {
			c.JSON(http.StatusBadRequest, &ErrorResponse{
				Type:     "/errors/missing-param-id",
				Title:    "id is empty",
				Status:   http.StatusBadRequest,
				Detail:   "id url param is missing from the request url. it is required for updating a key.",
				Instance: path,
			})

			return
		}

		data, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusInternalServerError, &ErrorResponse{
				Type:     "/errors/request-body-read",
				Title:    "request body reader error",
				Status:   http.StatusInternalServerError,
				Detail:   err.Error(),
				Instance: path,
			})
			return
		}

		uk := &key.UpdateKey{}
		err = json.Unmarshal(data, uk)
		if err != nil {
			c.JSON(http.StatusInternalServerError, &ErrorResponse{
				Type:     "/errors/json-unmarshal",
				Title:    "json unmarshaller error",
				Status:   http.StatusInternalServerError,
				Detail:   err.Error(),
				Instance: path,
			})
			return
		}

		resk, err := m.UpdateKey(id, uk)
		if err != nil {
			if _, ok := err.(ValidationError); ok {
				c.JSON(http.StatusBadRequest, &ErrorResponse{
					Type:     "/errors/validation",
					Title:    "key validation failed",
					Status:   http.StatusBadRequest,
					Detail:   err.Error(),
					Instance: path,
				})
				return
			}

			c.JSON(http.StatusInternalServerError, &ErrorResponse{
				Type:     "/errors/key-manager",
				Title:    "update key error",
				Status:   http.StatusInternalServerError,
				Detail:   err.Error(),
				Instance: path,
			})
			return
		}

		c.JSON(http.StatusOK, resk)
	}
}

func getDeleteKeyHandler(m KeyManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		path := "/api/key-management/keys/:id"
		if len(id) == 0 {
			c.JSON(http.StatusBadRequest, &ErrorResponse{
				Type:     "/errors/missing-param-id",
				Title:    "id is empty",
				Status:   http.StatusBadRequest,
				Detail:   "id url param is missing from the request url. it is required for deleting a key.",
				Instance: path,
			})

			return
		}

		err := m.DeleteKey(id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, &ErrorResponse{
				Type:     "/errors/key-manager",
				Title:    "key deletion error",
				Status:   http.StatusInternalServerError,
				Detail:   err.Error(),
				Instance: path,
			})
			return
		}

		c.Status(http.StatusOK)
	}
}
