package route

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/cenkalti/backoff/v4"
	goopenai "github.com/sashabaranov/go-openai"
	"go.uber.org/zap"

	"github.com/bricks-cloud/bricksllm/internal/event"
	"github.com/bricks-cloud/bricksllm/internal/key"
	"github.com/bricks-cloud/bricksllm/internal/provider"
	"github.com/bricks-cloud/bricksllm/internal/util"
)

type recorder interface {
	RecordEvent(e *event.Event) error
}

type CacheConfig struct {
	Enabled bool   `json:"enabled"`
	Ttl     string `json:"ttl"`
}

type Step struct {
	Retries       int               `json:"retries"`
	RetryInterval string            `json:"retryInterval"`
	Provider      string            `json:"provider"`
	RequestParams map[string]any    `json:"requestParams"`
	Params        map[string]string `json:"params"`
	Model         string            `json:"model"`
	Timeout       string            `json:"timeout"`
}

func ConvertToArrayOfStrings(input []any) []string {
	result := []string{}
	for _, input := range input {
		val, ok := input.(string)
		if ok {
			result = append(result, val)
			continue
		}
		return []string{}
	}

	return result
}

func ConvertToMapOfIntegers(input any) map[string]int {
	result := map[string]int{}

	parsed, ok := input.(map[string]any)
	if ok {
		for k, v := range parsed {
			parsedv, ok := v.(float64)
			if ok {
				result[k] = int(parsedv)
				continue
			}
			return map[string]int{}
		}
	}

	return result
}

func (s *Step) DecorateRequest(provider string, body []byte, isEmbedding bool) ([]byte, error) {
	if provider != "azure" {
		if isEmbedding {
			embeddingsReq := &goopenai.EmbeddingRequest{}

			err := json.Unmarshal(body, embeddingsReq)
			if err != nil {
				return nil, err
			}

			embeddingsReq.Model = goopenai.EmbeddingModel(s.Model)

			return json.Marshal(embeddingsReq)
		}
	}

	if !isEmbedding {
		completionReq := &goopenai.ChatCompletionRequest{}

		err := json.Unmarshal(body, completionReq)
		if err != nil {
			return nil, err
		}

		completionReq.Model = s.Model

		s.DecorateChatCompletionRequest(completionReq)

		return json.Marshal(completionReq)
	}

	return body, nil
}

func (s *Step) DecorateChatCompletionRequest(req *goopenai.ChatCompletionRequest) {
	if s == nil {
		return
	}

	req.Model = s.Model
	if val, ok := s.RequestParams["frequency_penalty"]; ok {
		if parsed, ok := val.(float64); ok {
			req.FrequencyPenalty = float32(parsed)
		}
	}

	if val, ok := s.RequestParams["max_tokens"]; ok {
		if parsed, ok := val.(float64); ok {
			req.MaxTokens = int(parsed)
		}
	}

	if val, ok := s.RequestParams["temperature"]; ok {
		if parsed, ok := val.(float64); ok {
			req.Temperature = float32(parsed)
		}
	}

	if val, ok := s.RequestParams["top_p"]; ok {
		if parsed, ok := val.(float64); ok {
			req.TopP = float32(parsed)
		}
	}

	if val, ok := s.RequestParams["n"]; ok {
		if parsed, ok := val.(float64); ok {
			req.N = int(parsed)
		}
	}

	if val, ok := s.RequestParams["stop"]; ok {
		if parsed, ok := val.([]any); ok {
			req.Stop = ConvertToArrayOfStrings(parsed)
		}
	}

	if val, ok := s.RequestParams["presence_penalty"]; ok {
		if parsed, ok := val.(float64); ok {
			req.PresencePenalty = float32(parsed)
		}
	}

	if val, ok := s.RequestParams["seed"]; ok {
		if parsed, ok := val.(float64); ok {
			seedInt := int(parsed)
			req.Seed = &seedInt
		}
	}

	if val, ok := s.RequestParams["logit_bias"]; ok {
		if parsed, ok := val.(map[string]any); ok {
			req.LogitBias = ConvertToMapOfIntegers(parsed)
		}
	}

	if val, ok := s.RequestParams["logprobs"]; ok {
		if parsed, ok := val.(bool); ok {
			req.LogProbs = parsed
		}
	}

	if val, ok := s.RequestParams["top_logprobs"]; ok {
		if parsed, ok := val.(float64); ok {
			req.TopLogProbs = int(parsed)
		}
	}
}

type Route struct {
	Id            string       `json:"id"`
	RetryStrategy string       `json:"retryStrategy"`
	RequestFormat string       `json:"requestFormat"`
	CreatedAt     int64        `json:"createdAt"`
	UpdatedAt     int64        `json:"updatedAt"`
	Name          string       `json:"name"`
	Path          string       `json:"path"`
	KeyIds        []string     `json:"keyIds"`
	Steps         []*Step      `json:"steps"`
	CacheConfig   *CacheConfig `json:"cacheConfig"`
}

func (r *Route) ValidateSettings(settings []*provider.Setting) bool {
	target := map[string]bool{}
	for _, s := range r.Steps {
		target[s.Provider] = true
	}

	source := map[string]bool{}
	for _, s := range settings {
		source[s.Provider] = true
	}

	for p := range target {
		if !source[p] {
			return false
		}
	}

	return true
}

func (r *Route) ShouldRunEmbeddings() bool {
	if len(r.RequestFormat) != 0 {
		return r.RequestFormat == "openai_embeddings"
	}

	if len(r.Steps) == 0 {
		return false
	}

	if r.Steps[0] == nil {
		return false
	}

	if strings.Contains(r.Steps[0].Model, "ada") {
		return true
	}

	return false
}

func InitializeBackoff(strategy string, dur time.Duration) backoff.BackOff {
	if strategy == "exponential" {
		b := backoff.NewExponentialBackOff()

		return b
	}

	return backoff.NewConstantBackOff(dur)
}

func (r *Route) RunStepsV2(req *Request, rec recorder, log *zap.Logger, kc *key.ResponseKey) (*Response, error) {
	if len(r.Steps) == 0 {
		return nil, errors.New("steps are empty")
	}

	body, err := io.ReadAll(req.Forwarded.Body)
	if err != nil {
		return nil, err
	}

	events := []*event.Event{}
	response := &Response{}

	for _, step := range r.Steps {
		dur := time.Second
		if len(step.RetryInterval) != 0 {
			parsed, err := time.ParseDuration(step.RetryInterval)
			if err != nil {
				return nil, err
			}

			dur = parsed
		}

		b := InitializeBackoff(r.RetryStrategy, dur)
		withRetries := backoff.WithMaxRetries(b, uint64(step.Retries))

		do := func() error {
			start := time.Now()

			evt := &event.Event{
				Id:            util.NewUuid(),
				CreatedAt:     time.Now().Unix(),
				Tags:          kc.Tags,
				KeyId:         kc.KeyId,
				Provider:      step.Provider,
				Method:        req.Forwarded.Method,
				Path:          req.Forwarded.URL.Path,
				Model:         step.Model,
				Action:        req.Action,
				Request:       []byte(`{}`),
				Response:      []byte(`{}`),
				CustomId:      req.Forwarded.Header.Get("X-CUSTOM-EVENT-ID"),
				UserId:        req.UserId,
				PolicyId:      req.PolicyId,
				RouteId:       r.Id,
				CorrelationId: req.CorrelationId,
			}

			defer func() {
				evt.LatencyInMs = int(time.Since(start).Milliseconds())
			}()

			events = append(events, evt)

			if kc.ShouldLogRequest {
				evt.Request = body
			}

			parsed, err := time.ParseDuration(step.Timeout)
			if err != nil {
				return err
			}

			bs, err := step.DecorateRequest(step.Provider, body, r.ShouldRunEmbeddings())
			if err != nil {
				return err
			}

			shouldNotCancel := false
			ctx, cancel := context.WithTimeout(context.Background(), parsed)
			defer func() {
				if !shouldNotCancel {
					cancel()
				}
			}()

			hreq, err := req.createHttpRequest(ctx, step.Provider, r.ShouldRunEmbeddings(), step.Params, bs)
			if err != nil {
				return err
			}

			res, err := req.Client.Do(hreq)
			if err != nil {
				return err
			}

			response.Provider = step.Provider
			response.Model = step.Model
			response.Response = res
			response.Cancel = cancel
			evt.Status = res.StatusCode

			if res.StatusCode != http.StatusOK {
				defer res.Body.Close()

				bytes, err := io.ReadAll(res.Body)
				if err != nil {
					return err
				}

				response.Data = bytes
				return errors.New("response is not okay")
			}

			if kc.ShouldLogResponse {
				evt.Response = body
			}

			if res.StatusCode == http.StatusOK {
				shouldNotCancel = true
			}

			return nil
		}

		notify := func(err error, t time.Duration) {
			log.Debug("error when requesting external api via route", zap.Error(err), zap.Duration("duration", t))
		}

		err := backoff.RetryNotify(do, withRetries, notify)
		if err == nil {
			break
		}
	}

	for idx, evt := range events {
		if idx != len(events)-1 {
			go func() {
				err := rec.RecordEvent(evt)
				if err != nil {
					log.Debug("error when recording event", zap.Error(err))
				}
			}()

			continue
		}
	}

	if response.Response != nil {
		return response, nil
	}

	return nil, errors.New("no responses")
}

func (r *Route) RunSteps(req *Request, rec recorder, log *zap.Logger) (*Response, error) {
	if len(r.Steps) == 0 {
		return nil, errors.New("steps are empty")
	}

	responses := []*http.Response{}
	cancelFuncs := []context.CancelFunc{}

	noResponses := false
	defer func() {
		for index, resp := range responses {
			if index != len(responses)-1 {
				if resp != nil {
					resp.Body.Close()
				}
			}
		}

		for index, cancel := range cancelFuncs {
			if index != len(cancelFuncs)-1 || noResponses {
				cancel()
			}
		}
	}()

	body, err := io.ReadAll(req.Forwarded.Body)
	if err != nil {
		return nil, err
	}

	var lastErr error
	stopStep := 0

	totalRetries := 0
	currentRetry := 0
	for _, step := range r.Steps {
		totalRetries += step.Retries
	}

	for idx, step := range r.Steps {
		resourceName := ""

		if step.Provider == "azure" {
			val, err := req.GetSettingValue("azure", "resourceName")
			if err != nil {
				return nil, err
			}

			resourceName = val
		}

		key, err := req.GetSettingValue(step.Provider, "apikey")
		if err != nil {
			return nil, err
		}

		parsed, err := time.ParseDuration(step.Timeout)
		if err != nil {
			return nil, err
		}

		retries := step.Retries
		if step.Retries == 0 {
			retries = 1
		}

		for retries > 0 {
			currentRetry++

			evt := &event.Event{
				Id:            util.NewUuid(),
				CreatedAt:     time.Now().Unix(),
				Provider:      step.Provider,
				Method:        req.Forwarded.Method,
				Path:          req.Forwarded.URL.Path,
				Model:         step.Model,
				Action:        req.Action,
				Request:       []byte(`{}`),
				Response:      []byte(`{}`),
				CustomId:      req.Forwarded.Header.Get("X-CUSTOM-EVENT-ID"),
				UserId:        req.UserId,
				PolicyId:      req.PolicyId,
				RouteId:       r.Id,
				CorrelationId: req.CorrelationId,
			}

			if req.Key != nil {
				evt.KeyId = req.Key.KeyId
				evt.Tags = req.Key.Tags

				if req.Key.ShouldLogRequest {
					evt.Request = req.Request
				}
			}

			url := buildRequestUrl(step.Provider, r.ShouldRunEmbeddings(), resourceName, step.Params)

			if len(url) == 0 {
				return nil, errors.New("only azure openai, openai chat completion and embeddings models are supported")
			}

			ctx, cancel := context.WithTimeout(context.Background(), parsed)
			cancelFuncs = append(cancelFuncs, cancel)

			selected := body

			if step.Provider == "openai" {
				if r.ShouldRunEmbeddings() {
					embeddingsReq := &goopenai.EmbeddingRequest{}

					err := json.Unmarshal(body, embeddingsReq)
					if err != nil {
						continue
					}

					embeddingsReq.Model = goopenai.EmbeddingModel(step.Model)

					selected, err = json.Marshal(embeddingsReq)
					if err != nil {
						continue
					}
				}

				if !r.ShouldRunEmbeddings() {
					completionReq := &goopenai.ChatCompletionRequest{}

					err := json.Unmarshal(body, completionReq)
					if err != nil {
						continue
					}

					step.DecorateChatCompletionRequest(completionReq)

					selected, err = json.Marshal(completionReq)
					if err != nil {
						continue
					}
				}
			}

			hreq, err := http.NewRequestWithContext(ctx, req.Forwarded.Method, url, io.NopCloser(bytes.NewReader(selected)))
			lastErr = err

			if err != nil {
				defer cancel()
				retries -= 1
				continue
			}

			setHttpRequestAuthHeader(step.Provider, hreq, key)

			for k := range req.Forwarded.Header {
				if strings.HasPrefix(strings.ToLower(k), "authorization") {
					continue
				}

				if strings.HasPrefix(strings.ToLower(k), "api-key") {
					continue
				}

				if strings.ToLower(k) == "accept-encoding" {
					continue
				}

				hreq.Header.Set(k, req.Forwarded.Header.Get(k))
			}

			res, err := req.Client.Do(hreq)
			lastErr = err
			stopStep = idx

			evt.LatencyInMs = int(time.Since(req.Start).Milliseconds())
			evt.Status = res.StatusCode

			if res.StatusCode != http.StatusOK && currentRetry != totalRetries {
				go func(input *event.Event) {
					err := rec.RecordEvent(input)
					if err != nil {
						log.Debug("error when recording event", zap.Error(err))
					}
				}(evt)
			}

			if err != nil {
				retries -= 1
				continue
			}

			responses = append(responses, res)

			if res.StatusCode != http.StatusOK {
				retries -= 1
				continue
			}

			break
		}

		if len(responses) > 0 && responses[len(responses)-1].StatusCode == http.StatusOK {
			break
		}
	}

	if errors.Is(lastErr, context.DeadlineExceeded) {
		return nil, lastErr
	}

	if len(responses) >= 1 {
		return &Response{
			Response: responses[len(responses)-1],
			Cancel:   cancelFuncs[len(cancelFuncs)-1],
			Provider: r.Steps[stopStep].Provider,
			Model:    r.Steps[stopStep].Model,
		}, nil
	}

	noResponses = true

	return nil, errors.New("no responses")
}

type Request struct {
	Settings      map[string]*provider.Setting
	Key           *key.ResponseKey
	Client        http.Client
	Forwarded     *http.Request
	Start         time.Time
	CustomId      string
	Request       []byte
	Response      []byte
	UserId        string
	PolicyId      string
	Action        string
	CorrelationId string
}

func (r *Request) GetSettingValue(provider string, param string) (string, error) {
	for _, setting := range r.Settings {
		if setting.Provider == provider {
			val, ok := setting.Setting[param]
			if ok {
				return val, nil
			}

			return "", errors.New(fmt.Sprintf("%s setting param: %s not found", provider, param))
		}
	}

	return "", errors.New(fmt.Sprintf("%s setting is not found", provider))
}

type Response struct {
	Provider string
	Model    string
	Data     []byte
	Cancel   context.CancelFunc
	Response *http.Response
}

func buildRequestUrl(provider string, runEmbeddings bool, resourceName string, params map[string]string) string {
	if provider == "openai" && runEmbeddings {
		return "https://api.openai.com/v1/embeddings"
	}

	if provider == "openai" && !runEmbeddings {
		return "https://api.openai.com/v1/chat/completions"
	}

	deploymentId := params["deploymentId"]
	apiVersion := params["apiVersion"]

	if provider == "azure" && runEmbeddings {
		return fmt.Sprintf("https://%s.openai.azure.com/openai/deployments/%s/embeddings?api-version=%s", resourceName, deploymentId, apiVersion)
	}

	if provider == "azure" && !runEmbeddings {
		return fmt.Sprintf("https://%s.openai.azure.com/openai/deployments/%s/chat/completions?api-version=%s", resourceName, deploymentId, apiVersion)
	}

	return ""
}

func setHttpRequestAuthHeader(provider string, req *http.Request, key string) {
	if provider == "azure" {
		req.Header.Set("api-key", key)
		return
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", key))
}

func (r *Request) createHttpRequest(ctx context.Context, provider string, runEmbeddings bool, params map[string]string, data []byte) (*http.Request, error) {
	resourceName := ""
	if provider == "azure" {
		val, err := r.GetSettingValue("azure", "resourceName")
		if err != nil {
			return nil, err
		}

		resourceName = val
	}

	key, err := r.GetSettingValue(provider, "apikey")
	if err != nil {
		return nil, err
	}

	url := buildRequestUrl(provider, runEmbeddings, resourceName, params)
	if len(url) == 0 {
		return nil, errors.New("request url is empty")
	}

	hreq, err := http.NewRequestWithContext(ctx, r.Forwarded.Method, url, io.NopCloser(bytes.NewReader(data)))

	if provider == "azure" {
		hreq.Header.Set("api-key", key)
	} else {
		hreq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", key))
	}

	for k := range r.Forwarded.Header {
		if strings.HasPrefix(strings.ToLower(k), "authorization") {
			continue
		}

		if strings.HasPrefix(strings.ToLower(k), "api-key") {
			continue
		}

		if strings.ToLower(k) == "accept-encoding" {
			continue
		}

		hreq.Header.Set(k, r.Forwarded.Header.Get(k))
	}

	return hreq, err
}
