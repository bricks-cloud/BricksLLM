package message

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/bricks-cloud/bricksllm/internal/event"
	"github.com/bricks-cloud/bricksllm/internal/key"
	"github.com/bricks-cloud/bricksllm/internal/provider/anthropic"
	"github.com/bricks-cloud/bricksllm/internal/provider/custom"
	"github.com/bricks-cloud/bricksllm/internal/provider/vllm"
	"github.com/bricks-cloud/bricksllm/internal/stats"
	"github.com/bricks-cloud/bricksllm/internal/user"
	"github.com/tidwall/gjson"
	"go.uber.org/zap"

	goopenai "github.com/sashabaranov/go-openai"
)

type anthropicEstimator interface {
	EstimateTotalCost(model string, promptTks, completionTks int) (float64, error)
	EstimateCompletionCost(model string, tks int) (float64, error)
	EstimatePromptCost(model string, tks int) (float64, error)
	Count(input string) int
}

type estimator interface {
	EstimateSpeechCost(input string, model string) (float64, error)
	EstimateChatCompletionPromptCostWithTokenCounts(r *goopenai.ChatCompletionRequest) (int, float64, error)
	EstimateEmbeddingsCost(r *goopenai.EmbeddingRequest) (float64, error)
	EstimateChatCompletionStreamCostWithTokenCounts(model, content string) (int, float64, error)
	EstimateCompletionCost(model string, tks int) (float64, error)
	EstimateTotalCost(model string, promptTks, completionTks int) (float64, error)
	EstimateEmbeddingsInputCost(model string, tks int) (float64, error)
	EstimateChatCompletionPromptTokenCounts(model string, r *goopenai.ChatCompletionRequest) (int, error)
}

type azureEstimator interface {
	EstimateChatCompletionStreamCostWithTokenCounts(model, content string) (int, float64, error)
	EstimateEmbeddingsCost(r *goopenai.EmbeddingRequest) (float64, error)
	EstimateCompletionCost(model string, tks int) (float64, error)
	EstimatePromptCost(model string, tks int) (float64, error)
	EstimateTotalCost(model string, promptTks, completionTks int) (float64, error)
	EstimateEmbeddingsInputCost(model string, tks int) (float64, error)
}

type vllmEstimator interface {
	EstimateCompletionPromptToken(r *vllm.CompletionRequest) int
	EstimateChatCompletionPromptToken(r *vllm.ChatRequest) int
	EstimateContentTokenCounts(model string, content string) int
}

type validator interface {
	Validate(k *key.ResponseKey, promptCost float64) error
}

type userValidator interface {
	Validate(u *user.User, promptCost float64) error
}

type keyManager interface {
	GetKeys(tags, keyIds []string, provider string) ([]*key.ResponseKey, error)
	UpdateKey(id string, uk *key.UpdateKey) (*key.ResponseKey, error)
}

type userManager interface {
	GetUsers(tags, keyIds, userIds []string, offset int, limit int) ([]*user.User, error)
	UpdateUser(id string, uu *user.UpdateUser) (*user.User, error)
}

type rateLimitManager interface {
	Increment(keyId string, timeUnit key.TimeUnit) error
	IncrementUser(id string, timeUnit key.TimeUnit) error
}

type accessCache interface {
	Set(key string, timeUnit key.TimeUnit) error
}

type userAccessCache interface {
	Set(key string, timeUnit key.TimeUnit) error
}

type Handler struct {
	recorder recorder
	log      *zap.Logger
	ae       anthropicEstimator
	e        estimator
	vllme    vllmEstimator
	aze      azureEstimator
	v        validator
	uv       userValidator
	km       keyManager
	um       userManager
	rlm      rateLimitManager
	ac       accessCache
	uac      userAccessCache
}

func NewHandler(r recorder, log *zap.Logger, ae anthropicEstimator, e estimator, vllme vllmEstimator, aze azureEstimator, v validator, uv userValidator, km keyManager, um userManager, rlm rateLimitManager, ac accessCache, uac accessCache) *Handler {
	return &Handler{
		recorder: r,
		log:      log,
		ae:       ae,
		e:        e,
		vllme:    vllme,
		aze:      aze,
		v:        v,
		uv:       uv,
		km:       km,
		um:       um,
		rlm:      rlm,
		ac:       ac,
	}
}

func (h *Handler) HandleEvent(m Message) error {
	stats.Incr("bricksllm.message.handler.handle_event.requests", nil, 1)

	e, ok := m.Data.(*event.Event)
	if !ok {
		stats.Incr("bricksllm.message.handler.handle_event.event_parsing_error", nil, 1)
		h.log.Info("message contains data that cannot be converted to event format", zap.Any("data", m.Data))
		return errors.New("message data cannot be parsed as event")
	}

	start := time.Now()

	err := h.recorder.RecordEvent(e)
	if err != nil {
		stats.Incr("bricksllm.message.handler.handle_event.record_event_error", nil, 1)
		h.log.Sugar().Debugf("error when publish in event: %v", err)
		return err
	}

	stats.Timing("bricksllm.message.handler.handle_event.record_event_latency", time.Since(start), nil, 1)
	stats.Incr("bricksllm.message.handler.handle_event.success", nil, 1)

	return nil
}

const (
	anthropicPromptMagicNum     int = 1
	anthropicCompletionMagicNum int = 4
)

func countTokensFromJson(bytes []byte, contentLoc string) (int, error) {
	content := getContentFromJson(bytes, contentLoc)
	return custom.Count(content)
}

func getContentFromJson(bytes []byte, contentLoc string) string {
	result := gjson.Get(string(bytes), contentLoc)
	content := ""

	if len(result.Str) != 0 {
		content += result.Str
	}

	if result.IsArray() {
		for _, val := range result.Array() {
			if len(val.Str) != 0 {
				content += val.Str
			}
		}
	}

	return content
}

type costLimitError interface {
	Error() string
	CostLimit()
}

type rateLimitError interface {
	Error() string
	RateLimit()
}

type expirationError interface {
	Error() string
	Reason() string
}

func (h *Handler) handleValidationResult(kc *key.ResponseKey, cost float64) error {
	err := h.v.Validate(kc, cost)

	if err != nil {
		stats.Incr("bricksllm.message.handler.handle_validation_result.handle_validation_result", nil, 1)

		if _, ok := err.(expirationError); ok {
			stats.Incr("bricksllm.message.handler.handle_validation_result.expiraton_error", nil, 1)

			tks, err := h.km.GetKeys(nil, []string{kc.KeyId}, "")
			if err != nil {
				stats.Incr("bricksllm.message.handler.handle_validation_result.get_keys_error", nil, 1)
			}

			if len(tks) == 1 {
				err := h.v.Validate(tks[0], cost)
				if err != nil {
					if xe, ok := err.(expirationError); ok {
						h.log.Debug("expiration error",
							zap.String("expired_reason", xe.Reason()),
							zap.String("key_id", kc.KeyId),
						)

						truePtr := true
						_, err = h.km.UpdateKey(kc.KeyId, &key.UpdateKey{
							Revoked:       &truePtr,
							RevokedReason: key.RevokedReasonExpired,
						})

						if err != nil {
							stats.Incr("bricksllm.message.handler.handle_validation_result.update_key_error", nil, 1)
							return err
						}

						return nil
					}
				}
			}
		}

		if _, ok := err.(rateLimitError); ok {
			stats.Incr("bricksllm.message.handler.handle_validation_result.rate_limit_error", nil, 1)

			err = h.ac.Set(kc.KeyId, kc.RateLimitUnit)
			if err != nil {
				stats.Incr("bricksllm.message.handler.handle_validation_result.set_rate_limit_error", nil, 1)
				return err
			}

			return nil
		}

		if _, ok := err.(costLimitError); ok {
			stats.Incr("bricksllm.message.handler.handle_validation_result.cost_limit_error", nil, 1)

			err = h.ac.Set(kc.KeyId, kc.CostLimitInUsdUnit)
			if err != nil {
				stats.Incr("bricksllm.message.handler.handle_validation_result.set_cost_limit_error", nil, 1)
				return err
			}

			return nil
		}

		return err
	}

	return nil
}

func (h *Handler) handleUserValidationResult(u *user.User, cost float64) error {
	err := h.uv.Validate(u, cost)

	if err != nil {
		stats.Incr("bricksllm.message.handler.handle_user_validation_result.validate_error", nil, 1)

		if _, ok := err.(expirationError); ok {
			stats.Incr("bricksllm.message.handler.handle_user_validation_result.expiraton_error", nil, 1)

			truePtr := true
			_, err = h.um.UpdateUser(u.Id, &user.UpdateUser{
				Revoked:       &truePtr,
				RevokedReason: key.RevokedReasonExpired,
			})

			if err != nil {
				stats.Incr("bricksllm.message.handler.handle_user_validation_result.update_user_error", nil, 1)
				return err
			}

			return nil
		}

		if _, ok := err.(rateLimitError); ok {
			stats.Incr("bricksllm.message.handler.handle_user_validation_result.rate_limit_error", nil, 1)

			err = h.uac.Set(u.Id, u.RateLimitUnit)
			if err != nil {
				stats.Incr("bricksllm.message.handler.handle_user_validation_result.set_rate_limit_error", nil, 1)
				return err
			}

			return nil
		}

		if _, ok := err.(costLimitError); ok {
			stats.Incr("bricksllm.message.handler.handle_user_validation_result.cost_limit_error", nil, 1)

			err = h.uac.Set(u.Id, u.CostLimitInUsdUnit)
			if err != nil {
				stats.Incr("bricksllm.message.handler.handle_user_validation_result.set_cost_limit_error", nil, 1)
				return err
			}

			return nil
		}

		return err
	}

	return nil
}

func (h *Handler) HandleEventWithRequestAndResponse(m Message) error {
	e, ok := m.Data.(*event.EventWithRequestAndContent)
	if !ok {
		stats.Incr("bricksllm.message.handler.handle_event_with_request_and_response.message_data_parsing_error", nil, 1)
		h.log.Debug("message contains data that cannot be converted to event with request and response format", zap.Any("data", m.Data))
		return errors.New("message data cannot be parsed as event with request and response")
	}

	if e.Key != nil && !e.Key.Revoked && e.Event != nil {
		err := h.decorateEvent(m)
		if err != nil {
			stats.Incr("bricksllm.message.handler.handle_event_with_request_and_response.decorate_event_error", nil, 1)
			h.log.Debug("error when decorating event", zap.Error(err))
		}

		var u *user.User

		if e.Event.CostInUsd != 0 {
			micros := int64(e.Event.CostInUsd * 1000000)
			err = h.recorder.RecordKeySpend(e.Event.KeyId, micros, e.Key.CostLimitInUsdUnit)
			if err != nil {
				stats.Incr("bricksllm.message.handler.handle_event_with_request_and_response.record_key_spend_error", nil, 1)
				h.log.Debug("error when recording key spend", zap.Error(err))
			}

			if len(e.Event.UserId) != 0 {
				us, err := h.um.GetUsers(e.Key.Tags, nil, []string{e.Event.UserId}, 0, 0)
				if err != nil {
					stats.Incr("bricksllm.message.handler.handle_event_with_request_and_response.get_users_error", nil, 1)
					h.log.Debug("error when getting users", zap.Error(err))
				}

				if len(us) == 1 {
					u = us[0]

					err = h.recorder.RecordUserSpend(u.Id, micros, u.CostLimitInUsdUnit)
					if err != nil {
						stats.Incr("bricksllm.message.handler.handle_event_with_request_and_response.record_user_spend_error", nil, 1)
						h.log.Debug("error when recording user spend", zap.Error(err))
					}
				}
			}
		}

		if len(e.Key.RateLimitUnit) != 0 {
			if err := h.rlm.Increment(e.Key.KeyId, e.Key.RateLimitUnit); err != nil {
				stats.Incr("bricksllm.message.handler.handle_event_with_request_and_response.rate_limit_increment_error", nil, 1)

				h.log.Debug("error when incrementing rate limit", zap.Error(err))
			}
		}

		if u != nil {
			if err := h.rlm.IncrementUser(u.Id, u.RateLimitUnit); err != nil {
				stats.Incr("bricksllm.message.handler.handle_event_with_request_and_response.rate_limit_increment_user_error", nil, 1)

				h.log.Debug("error when incrementing rate limit", zap.Error(err))
			}

			err = h.handleUserValidationResult(u, e.Event.CostInUsd)
			if err != nil {
				stats.Incr("bricksllm.message.handler.handle_event_with_request_and_response.handle_user_validation_result_error", nil, 1)
				h.log.Debug("error when handling user validation result", zap.Error(err))
			}
		}

		err = h.handleValidationResult(e.Key, e.Event.CostInUsd)
		if err != nil {
			stats.Incr("bricksllm.message.handler.handle_event_with_request_and_response.handle_validation_result_error", nil, 1)
			h.log.Debug("error when handling validation result", zap.Error(err))
		}

	}

	start := time.Now()
	err := h.recorder.RecordEvent(e.Event)
	if err != nil {
		h.log.Debug("error when recording an event", zap.Error(err))
		stats.Incr("bricksllm.message.handler.handle_event_with_request_and_response.record_event_error", nil, 1)
		return err
	}

	stats.Timing("bricksllm.message.handler.handle_event_with_request_and_response.latency", time.Since(start), nil, 1)
	stats.Incr("bricksllm.message.handler.handle_event_with_request_and_response.success", nil, 1)

	return nil
}

func (h *Handler) decorateEvent(m Message) error {
	stats.Incr("bricksllm.message.handler.decorate_event.request", nil, 1)

	e, ok := m.Data.(*event.EventWithRequestAndContent)
	if !ok {
		stats.Incr("bricksllm.message.handler.decorate_event.message_data_parsing_error", nil, 1)
		h.log.Debug("message contains data that cannot be converted to event with request and response format", zap.Any("data", m.Data))
		return errors.New("message data cannot be parsed as event with request and response")
	}

	if e.Event.Path == "/api/providers/openai/v1/audio/speech" {
		csr, ok := e.Request.(*goopenai.CreateSpeechRequest)
		if !ok {
			stats.Incr("bricksllm.message.handler.decorate_event.event_request_parsing_error", nil, 1)
			h.log.Debug("event contains request that cannot be converted to anthropic completion request", zap.Any("data", m.Data))
			return errors.New("event request data cannot be parsed as anthropic completion request")
		}

		if e.Event.Status == http.StatusOK {
			cost, err := h.e.EstimateSpeechCost(csr.Input, string(csr.Model))
			if err != nil {
				stats.Incr("bricksllm.message.handler.decorate_event.estimate_prompt_cost", nil, 1)
				h.log.Debug("event contains request that cannot be converted to anthropic completion request", zap.Error(err))
				return err
			}

			e.Event.CostInUsd = cost
		}
	}

	if e.Event.Path == "/api/providers/anthropic/v1/complete" {
		cr, ok := e.Request.(*anthropic.CompletionRequest)
		if !ok {
			stats.Incr("bricksllm.message.handler.decorate_event.event_request_parsing_error", nil, 1)
			h.log.Debug("event contains request that cannot be converted to anthropic completion request", zap.Any("data", m.Data))
			return errors.New("event request data cannot be parsed as anthropic completion request")
		}

		tks := h.ae.Count(cr.Prompt)
		tks += anthropicPromptMagicNum

		model := cr.Model
		cost, err := h.ae.EstimatePromptCost(model, tks)
		if err != nil {
			stats.Incr("bricksllm.message.handler.decorate_event.estimate_prompt_cost", nil, 1)
			h.log.Debug("event contains request that cannot be converted to anthropic completion request", zap.Error(err))
			return err
		}

		completiontks := h.ae.Count(e.Content)
		completiontks += anthropicCompletionMagicNum

		completionCost, err := h.ae.EstimateCompletionCost(model, completiontks)
		if err != nil {
			stats.Incr("bricksllm.message.handler.decorate_event.estimate_completion_cost_error", nil, 1)
			return err
		}

		e.Event.PromptTokenCount = tks

		e.Event.CompletionTokenCount = completiontks
		if e.Event.Status == http.StatusOK {
			e.Event.CostInUsd = completionCost + cost
		}
	}

	if e.Event.Path == "/api/providers/azure/openai/deployments/:deployment_id/chat/completions" {
		ccr, ok := e.Request.(*goopenai.ChatCompletionRequest)
		if !ok {
			stats.Incr("bricksllm.message.handler.decorate_event.event_request_parsing_error", nil, 1)
			h.log.Debug("event contains data that cannot be converted to azure openai completion request", zap.Any("data", m.Data))
			return errors.New("event request data cannot be parsed as azure openai completion request")
		}

		if ccr.Stream {
			tks, err := h.e.EstimateChatCompletionPromptTokenCounts("gpt-3.5-turbo", ccr)
			if err != nil {
				stats.Incr("bricksllm.message.decorate_event.estimate_chat_completion_prompt_token_counts_error", nil, 1)
				return err
			}

			cost, err := h.aze.EstimatePromptCost(e.Event.Model, tks)
			if err != nil {
				stats.Incr("bricksllm.message.decorate_event.estimate_prompt_cost_error", nil, 1)
				return err
			}

			completiontks, completionCost, err := h.aze.EstimateChatCompletionStreamCostWithTokenCounts(e.Event.Model, e.Content)
			if err != nil {
				stats.Incr("bricksllm.message.decorate_event.estimate_chat_completion_stream_cost_with_token_counts_error", nil, 1)
				return err
			}

			e.Event.PromptTokenCount = tks
			e.Event.CompletionTokenCount = completiontks

			if e.Event.Status == http.StatusOK {
				e.Event.CostInUsd = cost + completionCost
			}
		}
	}

	if e.Event.Path == "/api/providers/openai/v1/chat/completions" {
		ccr, ok := e.Request.(*goopenai.ChatCompletionRequest)
		if !ok {
			stats.Incr("bricksllm.message.handler.decorate_event.event_request_parsing_error", nil, 1)
			h.log.Debug("event contains data that cannot be converted to openai completion request", zap.Any("data", m.Data))
			return errors.New("event request data cannot be parsed as openai completion request")
		}

		if ccr.Stream {
			tks, cost, err := h.e.EstimateChatCompletionPromptCostWithTokenCounts(ccr)
			if err != nil {
				stats.Incr("bricksllm.message.handler.decorate_event.estimate_chat_completion_prompt_cost_with_token_counts", nil, 1)
				return err
			}

			completiontks, completionCost, err := h.e.EstimateChatCompletionStreamCostWithTokenCounts(e.Event.Model, e.Content)
			if err != nil {
				stats.Incr("bricksllm.message.handler.decorate_event.estimate_chat_completion_stream_cost_with_token_counts", nil, 1)
				return err
			}

			e.Event.PromptTokenCount = tks
			e.Event.CompletionTokenCount = completiontks
			if e.Event.Status == http.StatusOK {
				e.Event.CostInUsd = cost + completionCost
			}
		}
	}

	if e.Event.Path == "/api/providers/vllm/v1/chat/completions" {
		ccr, ok := e.Request.(*vllm.ChatRequest)
		if !ok {
			stats.Incr("bricksllm.message.handler.decorate_event.event_request_parsing_error", nil, 1)
			h.log.Debug("event contains data that cannot be converted to vllm chat completion request", zap.Any("data", m.Data))
			return errors.New("event request data cannot be parsed as vllm chat completion request")
		}

		if ccr.Stream {
			e.Event.PromptTokenCount = h.vllme.EstimateChatCompletionPromptToken(ccr)
			e.Event.CompletionTokenCount = h.vllme.EstimateContentTokenCounts(e.Event.Model, e.Content)
		}
	}

	if e.Event.Path == "/api/providers/vllm/v1/completions" {
		cr, ok := e.Request.(*vllm.CompletionRequest)
		if !ok {
			stats.Incr("bricksllm.message.handler.decorate_event.event_request_parsing_error", nil, 1)
			h.log.Debug("event contains data that cannot be converted to vllm completion request", zap.Any("data", m.Data))
			return errors.New("event request data cannot be parsed as vllm completion request")
		}

		if cr.Stream {
			e.Event.PromptTokenCount = h.vllme.EstimateCompletionPromptToken(cr)
			e.Event.CompletionTokenCount = h.vllme.EstimateContentTokenCounts(e.Event.Model, e.Content)
		}
	}

	if e.Event.Path == "/api/providers/deepinfra/v1/chat/completions" {
		ccr, ok := e.Request.(*vllm.ChatRequest)
		if !ok {
			stats.Incr("bricksllm.message.handler.decorate_event.event_request_parsing_error", nil, 1)
			h.log.Debug("event contains data that cannot be converted to deepinfra chat completion request", zap.Any("data", m.Data))
			return errors.New("event request data cannot be parsed as deepinfra chat completion request")
		}

		if ccr.Stream {
			e.Event.PromptTokenCount = h.vllme.EstimateChatCompletionPromptToken(ccr)
			e.Event.CompletionTokenCount = h.vllme.EstimateContentTokenCounts(e.Event.Model, e.Content)
		}
	}

	if e.Event.Path == "/api/providers/deepinfra/v1/completions" {
		cr, ok := e.Request.(*vllm.CompletionRequest)
		if !ok {
			stats.Incr("bricksllm.message.handler.decorate_event.event_request_parsing_error", nil, 1)
			h.log.Debug("event contains data that cannot be converted to deepinfra completion request", zap.Any("data", m.Data))
			return errors.New("event request data cannot be parsed as deepinfra completion request")
		}

		if cr.Stream {
			e.Event.PromptTokenCount = h.vllme.EstimateCompletionPromptToken(cr)
			e.Event.CompletionTokenCount = h.vllme.EstimateContentTokenCounts(e.Event.Model, e.Content)
		}
	}

	if strings.HasPrefix(e.Event.Path, "/api/custom/providers/:provider") && e.RouteConfig != nil {
		body, ok := e.Request.([]byte)
		if !ok {
			stats.Incr("bricksllm.message.handler.decorate_event.event_request_custom_provider_parsing_error", nil, 1)
			h.log.Debug("event contains request that cannot be converted to bytes", zap.Any("data", m.Data))
			return errors.New("event request data cannot be parsed as anthropic completion request")
		}

		tks, err := countTokensFromJson(body, e.RouteConfig.RequestPromptLocation)
		if err != nil {
			stats.Incr("bricksllm.message.handler.decorate_event.count_tokens_from_json_error", nil, 1)

			return err
		}

		e.Event.PromptTokenCount = tks

		result := gjson.Get(string(body), e.RouteConfig.StreamLocation)
		if result.IsBool() {
			completiontks, err := custom.Count(e.Content)
			if err != nil {
				stats.Incr("bricksllm.message.handler.decorate_event.custom_count_error", nil, 1)
				return err
			}

			e.Event.CompletionTokenCount = completiontks
		}

		if !result.IsBool() {
			content, ok := e.Response.([]byte)
			if !ok {
				stats.Incr("bricksllm.message.handler.decorate_event.event_response_custom_provider_parsing_error", nil, 1)
				h.log.Debug("event contains response that cannot be converted to bytes", zap.Any("data", m.Data))
				return errors.New("event response data cannot be converted to bytes")
			}

			completiontks, err := countTokensFromJson(content, e.RouteConfig.ResponseCompletionLocation)
			if err != nil {
				stats.Incr("bricksllm.message.handler.decorate_event.count_tokens_from_json_error", nil, 1)
				return err
			}

			e.Event.CompletionTokenCount = completiontks
		}
	}

	return nil
}
