package route

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

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

func (r *Route) ShouldRunEmbeddings() bool {
	if len(r.Steps) == 0 {
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
	defer func() {
		for index, resp := range responses {
			if index != len(responses)-1 {
				resp.Body.Close()
			}
		}
	}()

	for _, step := range r.Steps {
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

		var final *http.Response
		for retries > 0 {
			url := buildRequestUrl(step.Provider, r.ShouldRunEmbeddings(), resourceName, step.Params)

			ctx, cancel := context.WithTimeout(context.Background(), parsed)
			defer cancel()
			hreq, err := http.NewRequestWithContext(ctx, req.Forwarded.Method, url, req.Forwarded.Body)
			if err != nil {
				return nil, err
			}

			setHttpRequestAuthHeader(step.Provider, hreq, key)

			for k := range req.Forwarded.Header {
				if !strings.HasPrefix(strings.ToLower(k), "x") {
					hreq.Header.Set(k, req.Forwarded.Header.Get(k))
				}
			}

			res, err := req.Client.Do(hreq)
			if err != nil {
				return nil, err
			}

			responses = append(responses, res)
			final = res

			if res.StatusCode != http.StatusOK {
				retries -= 1
				continue
			}
			break
		}

		if final.StatusCode == http.StatusOK {
			return &Response{
				Response: final,
				Provider: step.Provider,
				Model:    step.Model,
			}, nil
		}
	}

	if len(responses) >= 1 {
		return &Response{
			Response: responses[len(responses)-1],
			Provider: r.Steps[len(r.Steps)-1].Provider,
			Model:    r.Steps[len(r.Steps)-1].Model,
		}, nil
	}

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
