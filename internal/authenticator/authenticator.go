package auth

import (
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"strings"

	"github.com/bricks-cloud/bricksllm/internal/encrypter"
	internal_errors "github.com/bricks-cloud/bricksllm/internal/errors"
	"github.com/bricks-cloud/bricksllm/internal/stats"

	"github.com/bricks-cloud/bricksllm/internal/key"
	"github.com/bricks-cloud/bricksllm/internal/provider"
	"github.com/bricks-cloud/bricksllm/internal/route"
)

type providerSettingsManager interface {
	GetSetting(id string) (*provider.Setting, error)
}

type routesManager interface {
	GetRouteFromMemDb(path string) *route.Route
}

type keyMemStorage interface {
	GetKey(hash string) *key.ResponseKey
}

type keyStorage interface {
	GetKeyByHash(hash string) (*key.ResponseKey, error)
}

type Authenticator struct {
	psm providerSettingsManager
	kms keyMemStorage
	rm  routesManager
	ks  keyStorage
}

func NewAuthenticator(psm providerSettingsManager, kms keyMemStorage, rm routesManager, ks keyStorage) *Authenticator {
	return &Authenticator{
		psm: psm,
		kms: kms,
		rm:  rm,
		ks:  ks,
	}
}

func getApiKey(req *http.Request) (string, error) {
	list := []string{
		req.Header.Get("x-api-key"),
		req.Header.Get("api-key"),
	}

	split := strings.Split(req.Header.Get("Authorization"), " ")

	if len(split) >= 2 {
		list = append(list, split[1])
	}

	for _, key := range list {
		if len(key) != 0 {
			return key, nil
		}
	}

	return "", internal_errors.NewAuthError("api key not found")
}

func rewriteHttpAuthHeader(req *http.Request, setting *provider.Setting) error {
	uri := req.URL.RequestURI()
	if strings.HasPrefix(uri, "/api/routes") {
		return nil
	}

	apiKey := setting.GetParam("apikey")

	if len(apiKey) == 0 {
		return errors.New("api key is empty in provider setting")
	}

	if strings.HasPrefix(uri, "/api/providers/anthropic") {
		req.Header.Set("x-api-key", setting.GetParam("apikey"))
		return nil
	}

	if strings.HasPrefix(uri, "/api/providers/azure") {
		req.Header.Set("api-key", setting.GetParam("apikey"))
		return nil
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", setting.GetParam("apikey")))

	return nil
}

func (a *Authenticator) canKeyAccessCustomRoute(path string, keyId string) error {
	trimed := strings.TrimPrefix(path, "/api/routes")
	rc := a.rm.GetRouteFromMemDb(trimed)
	if rc == nil {
		return internal_errors.NewNotFoundError("route not found")
	}

	for _, kid := range rc.KeyIds {
		if kid == keyId {
			return nil
		}
	}

	return internal_errors.NewAuthError("not authorized")
}

func (a *Authenticator) getProviderSettingsThatCanAccessCustomRoute(path string, settings []*provider.Setting) []*provider.Setting {
	trimed := strings.TrimPrefix(path, "/api/routes")
	rc := a.rm.GetRouteFromMemDb(trimed)

	selected := []*provider.Setting{}
	if rc == nil {
		return []*provider.Setting{}
	}

	target := map[string]bool{}
	for _, s := range rc.Steps {
		target[s.Provider] = true
	}

	source := map[string]*provider.Setting{}
	for _, s := range settings {
		source[s.Provider] = s
	}

	for p := range target {
		if source[p] == nil {
			return []*provider.Setting{}
		}

		selected = append(selected, source[p])
	}

	return selected
}

func canAccessPath(provider string, path string) bool {
	if provider == "openai" && !strings.HasPrefix(path, "/api/providers/openai") {
		return false
	}

	if provider == "azure" && !strings.HasPrefix(path, "/api/providers/azure/openai") {
		return false
	}

	if provider == "anthropic" && !strings.HasPrefix(path, "/api/providers/anthropic") {
		return false
	}

	return true
}

type notFoundError interface {
	Error() string
	NotFound()
}

func (a *Authenticator) AuthenticateHttpRequest(req *http.Request) (*key.ResponseKey, []*provider.Setting, error) {
	raw, err := getApiKey(req)
	if err != nil {
		return nil, nil, err
	}

	hash := encrypter.Encrypt(raw)

	key := a.kms.GetKey(hash)
	if key != nil {
		stats.Incr("bricksllm.authenticator.authenticate_http_request.found_key_from_memdb", nil, 1)
	}

	if key == nil {
		key, err = a.ks.GetKeyByHash(hash)
		if err != nil {
			_, ok := err.(notFoundError)
			if ok {
				return nil, nil, internal_errors.NewAuthError("key not found")
			}

			return nil, nil, err
		}

		if key != nil {
			stats.Incr("bricksllm.authenticator.authenticate_http_request.found_key_from_db", nil, 1)
		}
	}

	if key == nil {
		return nil, nil, internal_errors.NewAuthError("key not found")
	}

	if key.Revoked {
		return nil, nil, internal_errors.NewAuthError("not authorized")
	}

	if strings.HasPrefix(req.URL.Path, "/api/routes") {
		err = a.canKeyAccessCustomRoute(req.URL.Path, key.KeyId)
		if err != nil {
			return nil, nil, err
		}
	}

	settingIds := key.GetSettingIds()
	allSettings := []*provider.Setting{}
	selected := []*provider.Setting{}
	for _, settingId := range settingIds {
		setting, err := a.psm.GetSetting(settingId)
		if err != nil {
			return nil, nil, err
		}

		if canAccessPath(setting.Provider, req.URL.Path) {

			selected = append(selected, setting)
		}

		allSettings = append(allSettings, setting)
	}

	if strings.HasPrefix(req.URL.Path, "/api/routes") {
		selected = a.getProviderSettingsThatCanAccessCustomRoute(req.URL.Path, allSettings)

		if len(selected) == 0 {
			return nil, nil, internal_errors.NewAuthError("provider settings associated with the key are not compatible with the route")
		}
	}

	if len(selected) != 0 {
		used := selected[0]
		if key.RotationEnabled {
			used = selected[rand.Intn(len(selected))]
		}

		err := rewriteHttpAuthHeader(req, used)
		if err != nil {
			return nil, nil, err
		}

		return key, selected, nil
	}

	return nil, nil, internal_errors.NewAuthError("provider setting not found")
}
