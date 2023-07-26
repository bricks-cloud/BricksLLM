package web

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"reflect"
	"strings"

	"github.com/bricks-cloud/atlas/config"
	"github.com/bricks-cloud/atlas/internal/client/openai"
	"github.com/bricks-cloud/atlas/internal/logger"
	"github.com/bricks-cloud/atlas/internal/util"
	"github.com/gin-gonic/gin"
)

type WebServer struct {
	server       *http.Server
	openAiClinet *openai.OpenAiClient
}

func NewWebServer(c *config.Config, lg logger.Logger) (*WebServer, error) {
	port := c.Server.Port

	router := gin.Default()
	openAiClient := openai.NewOpenAiClient(c.OpenAiConfig.ApiCredential)

	for _, rc := range c.Routes {
		r, err := NewRoute(rc, openAiClient, lg)
		if err != nil {
			return nil, errors.New("errors with setting up the web server")
		}

		router.POST(rc.Path, r.newRequestHandler())
	}

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: router,
	}

	return &WebServer{
		server: srv,
	}, nil
}

func (w *WebServer) Run() {
	go func() {
		if err := w.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()
}

func (w *WebServer) Shutdown(ctx context.Context) error {
	if err := w.server.Shutdown(ctx); err != nil {
		return err
	}

	return nil
}

type corsConfig interface {
	Enabled() bool
	GetAllowedOrigins() []string
	GetAllowedCredentials() bool
}

type keyAuthConfig interface {
	Enabled() bool
	GetKey() string
}

type Route struct {
	logger            logger.Logger
	path              string
	openAiRouteConfig *config.OpenAiRouteConfig
	openAiClient      openai.OpenAiClient
	corsConfig        corsConfig
	keyAuthConfig     keyAuthConfig
	dataSchema        reflect.Type
}

func NewRoute(rc *config.RouteConfig, openAiClient openai.OpenAiClient, lg logger.Logger) (*Route, error) {
	structSchema, err := newInputStruct(rc.Input)
	if err != nil {
		return nil, err
	}

	return &Route{
		logger:            lg,
		path:              rc.Path,
		openAiClient:      openAiClient,
		openAiRouteConfig: rc.OpenAiConfig,
		corsConfig:        rc.CorsConfig,
		keyAuthConfig:     rc.KeyAuthConfig,
		dataSchema:        structSchema,
	}, nil
}

func (r *Route) newRequestHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.Request.Header.Get("x-api-key")
		if r.keyAuthConfig.Enabled() {
			if r.keyAuthConfig.GetKey() != apiKey {
				c.Status(http.StatusUnauthorized)
				return
			}
		}

		jsonData, err := ioutil.ReadAll(c.Request.Body)
		if err != nil {
			c.IndentedJSON(http.StatusInternalServerError, err.Error())
			return
		}

		origin := c.Request.Header.Get("origin")
		if r.corsConfig.Enabled() {
			for _, route := range r.corsConfig.GetAllowedOrigins() {
				if origin == route || route == "*" {
					c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
					if r.corsConfig.GetAllowedCredentials() {
						c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
					}
					break
				}
			}
		}

		data := reflect.New(r.dataSchema)
		err = json.Unmarshal(jsonData, data.Interface())
		if err != nil {
			c.IndentedJSON(http.StatusInternalServerError, err.Error())
			return
		}

		prompts, err := r.populatePrompts(data)
		if err != nil {
			c.IndentedJSON(http.StatusInternalServerError, err.Error())
			return
		}

		res, err := r.openAiClient.Send(r.openAiRouteConfig, prompts)
		if err != nil {
			c.IndentedJSON(http.StatusInternalServerError, err.Error())
			return
		}

		r.logger.Infow("successful request")

		c.IndentedJSON(http.StatusOK, res)
	}
}

func (r *Route) populatePrompts(data reflect.Value) ([]*config.OpenAiPrompt, error) {
	populated := []*config.OpenAiPrompt{}

	for _, prompt := range r.openAiRouteConfig.Prompts {
		result, err := populateVariablesInPromptTemplate(prompt.Content, data)

		if err != nil {
			return nil, err
		}

		populated = append(populated, &config.OpenAiPrompt{
			Role:    prompt.Role,
			Content: result,
		})
	}

	return populated, nil
}

func newInputStruct(input map[string]config.InputValue) (reflect.Type, error) {
	structFields := []reflect.StructField{}
	for field, iv := range input {
		if iv.DataType == config.ObjectDataType {
			if iv.Properties == nil {
				return nil, errors.New("input object properties field can not be empty")
			}

			parsed := map[string]config.InputValue{}
			bs, err := json.Marshal(iv.Properties)
			if err != nil {
				return nil, fmt.Errorf("error when marshalling input object properties: %v", err)
			}

			err = json.Unmarshal(bs, &parsed)
			if err != nil {
				return nil, fmt.Errorf("error when unmarshalling input object properties: %v", err)
			}

			t, err := newInputStruct(parsed)
			if err != nil {
				return t, err
			}

			structFields = append(structFields, reflect.StructField{
				Name: strings.Title(field),
				Type: t,
				Tag:  reflect.StructTag(fmt.Sprintf(`json:"%s"`, field)),
			})
			continue
		}

		if iv.DataType == config.ArrayDataType {
			if iv.Items == nil {
				return nil, errors.New("input array items field can not be empty")
			}

			parsed := map[string]config.InputValue{}
			bs, err := json.Marshal(iv.Items)
			if err != nil {
				return nil, fmt.Errorf("error when marshalling input value: %v", err)
			}

			err = json.Unmarshal(bs, &parsed)
			if err != nil {
				return nil, fmt.Errorf("error when unmarshalling input fields: %v", err)
			}

			t, err := newInputStruct(parsed)
			if err != nil {
				return t, err
			}

			structFields = append(structFields, reflect.StructField{
				Name: strings.Title(field),
				Type: reflect.SliceOf(t),
				Tag:  reflect.StructTag(fmt.Sprintf(`json:"%s"`, field)),
			})
			continue
		}

		structFields = append(structFields, reflect.StructField{
			Name: strings.Title(field),
			Type: reflect.TypeOf(getType(iv.DataType)),
			Tag:  reflect.StructTag(fmt.Sprintf(`json:"%s"`, field)),
		})
	}

	t := reflect.StructOf(structFields)

	return t, nil
}

func getType(t config.DataType) interface{} {
	switch t {
	case config.BooleanDataType:
		return true
	case config.NumberDataType:
		return 1.11111
	default:
		return ""
	}
}

func populateVariablesInPromptTemplate(propmtContent string, val reflect.Value) (string, error) {
	populated := propmtContent
	variableMap := util.GetVariableMap(propmtContent)

	for old, reference := range variableMap {
		val, err := accessValueFromDataStruct(val, reference)
		if err != nil {
			return "", err
		}

		bs, err := json.Marshal(val)

		populated = strings.ReplaceAll(populated, old, string(bs))

	}

	return populated, nil
}

func accessValueFromDataStruct(val reflect.Value, reference string) (interface{}, error) {
	ele := val.Elem()
	if ele.IsZero() {
		return nil, errors.New("data struct is empty")
	}

	parts := strings.Split(reference, ".")
	if len(parts) == 0 {
		return nil, errors.New("reference is empty")
	}

	if len(parts) == 1 {
		inner := ele.FieldByName(strings.Title(parts[0]))
		if inner.IsZero() {
			return nil, errors.New("inner data struct is empty")
		}

		return inner.Interface(), nil
	}

	inner := ele.FieldByName(strings.Title(parts[0]))
	for index, part := range parts {
		if index == 0 {
			continue
		}

		if index != len(parts)-1 {
			inner = inner.FieldByName(strings.Title(part))
			if inner.IsZero() {
				return nil, fmt.Errorf("referenced data struct is empty: %s", part)
			}

			continue
		}
	}

	return inner.Interface(), nil
}
