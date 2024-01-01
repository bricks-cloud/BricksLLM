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

	goopenai "github.com/sashabaranov/go-openai"

	"github.com/bricks-cloud/bricksllm/internal/key"
	"github.com/bricks-cloud/bricksllm/internal/provider"
)

type CacheConfig struct {
	Enabled bool   `json:"enabled"`
	Ttl     string `json:"ttl"`
}

type Step struct {
	Retries  int               `json:"retries"`
	Provider string            `json:"provider"`
	Params   map[string]string `json:"params"`
	Model    string            `json:"model"`
	Timeout  string            `json:"timeout"`
}

type Route struct {
	Id          string       `json:"id"`
	CreatedAt   int64        `json:"createdAt"`
	UpdatedAt   int64        `json:"updatedAt"`
	Name        string       `json:"name"`
	Path        string       `json:"path"`
	KeyIds      []string     `json:"keyIds"`
	Steps       []*Step      `json:"steps"`
	CacheConfig *CacheConfig `json:"cacheConfig"`
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

func (r *Route) RunSteps(req *Request) (*Response, error) {
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

					embeddingsReq.Model = goopenai.AdaEmbeddingV2

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

					completionReq.Model = step.Model

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

				hreq.Header.Set(k, req.Forwarded.Header.Get(k))
			}

			res, err := req.Client.Do(hreq)
			lastErr = err
			stopStep = idx

			if err != nil {
				retries -= 1
				continue
			}

			if res.StatusCode != http.StatusOK {
				retries -= 1
				continue
			}

			responses = append(responses, res)

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
	Settings  map[string]*provider.Setting
	Key       *key.ResponseKey
	Client    http.Client
	Forwarded *http.Request
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
