package web

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/bricks-cloud/bricksllm/config"
	"github.com/bricks-cloud/bricksllm/internal/client/openai"
	"github.com/bricks-cloud/bricksllm/internal/logger"
	"github.com/bricks-cloud/bricksllm/internal/util"
	"github.com/gin-gonic/gin"
)

type WebServer struct {
	server       *http.Server
	logger       logger.Logger
	openAiClinet *openai.OpenAiClient
}

func NewWebServer(c *config.Config, lg logger.Logger, mode string) (*WebServer, error) {
	port := c.Server.Port

	router := gin.New()
	openAiClient := openai.NewOpenAiClient(c.OpenAiConfig.ApiCredential)

	for _, rc := range c.Routes {
		r, err := NewRoute(rc, openAiClient, lg, c.LoggerConfig, mode)
		if err != nil {
			return nil, errors.New("errors with setting up the web server")
		}

		router.POST(rc.Path, r.newRequestHandler())

		lg.Infof("%s is set up", rc.Path)
	}

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: router,
	}

	return &WebServer{
		logger: lg,
		server: srv,
	}, nil
}

func (w *WebServer) Run() {
	go func() {
		if err := w.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			w.logger.Fatalf("proxy server listen: %s\n", err)
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

type apiLoggerConfig interface {
	GetHideIp() bool
	GetHideHeaders() bool
}

type llmLoggerConfig interface {
	GetHideHeaders() bool
	GetHideResponseContent() bool
	GetHidePromptContent() bool
}

type Route struct {
	mode              string
	logger            logger.Logger
	apiLoggerConfig   apiLoggerConfig
	llmLoggerConfig   llmLoggerConfig
	provider          string
	path              string
	protocol          string
	openAiRouteConfig *config.OpenAiRouteConfig
	openAiClient      openai.OpenAiClient
	corsConfig        corsConfig
	keyAuthConfig     keyAuthConfig
	dataSchema        reflect.Type
}

func NewRoute(rc *config.RouteConfig, openAiClient openai.OpenAiClient, lg logger.Logger, lc *config.LoggerConfig, mode string) (*Route, error) {
	structSchema, err := newInputStruct(rc.Input)
	if err != nil {
		return nil, err
	}

	return &Route{
		logger:            lg,
		apiLoggerConfig:   lc.Api,
		llmLoggerConfig:   lc.Llm,
		mode:              mode,
		provider:          string(rc.Provider),
		protocol:          string(rc.Protocol),
		path:              rc.Path,
		openAiClient:      openAiClient,
		openAiRouteConfig: rc.OpenAiConfig,
		corsConfig:        rc.CorsConfig,
		keyAuthConfig:     rc.KeyAuthConfig,
		dataSchema:        structSchema,
	}, nil
}

func (r *Route) newApiMessage() *logger.ApiMessage {
	am := logger.NewApiMessage()
	am.SetCreatedAt(time.Now().Unix())
	am.SetPath(r.path)
	am.SetProtocol(r.protocol)
	return am
}

func (r *Route) newLlmMessage() *logger.LlmMessage {
	lm := logger.NewLlmMessage()
	lm.SetProvider(r.provider)
	lm.SetCreatedAt(time.Now().Unix())
	lm.SetRequestModel(string(r.openAiRouteConfig.Model))
	return lm
}

func newErrMessage() *logger.ErrorMessage {
	return logger.NewErrorMessage()
}

const (
	apiKeyHeader string = "X-Api-Key"
	forwardedFor string = "X-Forwarded-For"
)

func readUserIP(r *http.Request) string {
	address := r.Header.Get(forwardedFor)
	parts := strings.Split(address, ",")
	ip := ""

	if len(parts) > 0 {
		ip = parts[0]
	}

	if len(ip) == 0 {
		ip = r.RemoteAddr
	}

	return ip
}

type openAiError interface {
	Error() string
	StatusCode() int
}

func (r *Route) newRequestHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		am := r.newApiMessage()
		lm := r.newLlmMessage()
		instanceId := util.NewUuid()
		am.SetInstanceId(instanceId)
		lm.SetInstanceId(instanceId)

		em := newErrMessage()
		em.SetInstanceId(instanceId)
		var err error

		var proxyStart time.Time

		if c.Request != nil {
			ip := net.ParseIP(readUserIP(c.Request))
			if ip != nil {
				am.SetClientIp(ip.String())
			}

			am.SetRequestBodySize(c.Request.ContentLength)
			am.SetRequestHeaders(util.FilterHeaders(c.Request.Header, []string{
				apiKeyHeader,
				forwardedFor,
			}))

		}

		start := time.Now()
		defer func() {
			now := time.Now()
			total := now.Sub(start).Milliseconds()

			errExists := false
			if err != nil {
				em.SetCreatedAt(now.Unix())
				errExists = true
			}

			am.SetTotalLatency(total)
			if !proxyStart.IsZero() {
				latency := now.Sub(proxyStart).Milliseconds()
				am.SetProxyLatency(latency)
				lm.SetLatency(latency)
				am.SetBricksLlmLatency(total - am.GetProxyLatency())
			}

			am.SetResponseHeaders(c.Writer.Header())
			am.SetResponseStatus(c.Writer.Status())
			am.ModifyFileds(r.apiLoggerConfig)
			lm.ModifyFileds(r.llmLoggerConfig)

			if err != nil {
				em.SetMessage(err.Error())
			}

			if r.mode == "production" {
				r.logger.Infow("api message", "context", am)
				r.logger.Infow("llm message", "context", lm)
				r.logger.Debugw("error message", "context", em)
				return
			}

			data, err := json.MarshalIndent(em, "", "    ")
			if errExists && err == nil {
				r.logger.Debug(em.DevLogContext(), "\n", string(data))
			}

			data, err = json.MarshalIndent(am, "", "    ")
			if err == nil {
				r.logger.Info(am.DevLogContext(), "\n", string(data))
			}

			data, err = json.MarshalIndent(lm, "", "    ")
			if err == nil {
				r.logger.Info(lm.DevLogContext(), "\n", string(data))
			}
		}()

		apiKey := c.Request.Header.Get(apiKeyHeader)
		if r.keyAuthConfig.Enabled() {
			if r.keyAuthConfig.GetKey() != apiKey {
				err = errors.New("unauthorized http request with mismatched api key")
				c.Status(http.StatusUnauthorized)
				return
			}
		}

		jsonData, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusInternalServerError, err.Error())
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
			c.JSON(http.StatusInternalServerError, err.Error())
			return
		}

		prompts, err := r.populatePrompts(data)
		if err != nil {
			c.JSON(http.StatusInternalServerError, err.Error())
			return
		}

		proxyStart = time.Now()
		res, err := r.openAiClient.Send(r.openAiRouteConfig, prompts, lm)
		am.SetResponseCreatedAt(time.Now().Unix())
		if err != nil {
			if oae, ok := err.(openAiError); ok {
				c.JSON(oae.StatusCode(), err.Error())
				return
			}

			c.JSON(http.StatusInternalServerError, err.Error())
			return
		}

		lm.SetCompletionTokens(res.Usage.CompletionTokens)
		lm.SetPromptTokens(res.Usage.PromptTokens)
		lm.SetTotalTokens(res.Usage.TotalTokens)
		lm.SetResponseId(res.Id)

		resData, err := json.Marshal(res)
		if err != nil {
			c.JSON(http.StatusInternalServerError, err.Error())
			return
		}

		size, err := c.Writer.Write(resData)
		if err != nil {
			c.JSON(http.StatusInternalServerError, err.Error())
			return
		}

		am.SetResponseBodySize(int64(size))
		c.Status(http.StatusOK)
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

		stringified := fmt.Sprint(val)
		if len(stringified) == 0 {
			return "", fmt.Errorf("input value is empty: %v", err)
		}

		populated = strings.ReplaceAll(populated, old, stringified)
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

		inner = inner.FieldByName(strings.Title(part))
		if inner.IsZero() {
			return nil, fmt.Errorf("referenced data struct is empty: %s", part)
		}

		continue
	}

	return inner.Interface(), nil
}
