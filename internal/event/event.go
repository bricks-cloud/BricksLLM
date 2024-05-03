package event

import (
	"fmt"
	"strings"

	internal_errors "github.com/bricks-cloud/bricksllm/internal/errors"
)

type Event struct {
	Id                   string   `json:"id"`
	CreatedAt            int64    `json:"created_at"`
	Tags                 []string `json:"tags"`
	KeyId                string   `json:"key_id"`
	CostInUsd            float64  `json:"cost_in_usd"`
	Provider             string   `json:"provider"`
	Model                string   `json:"model"`
	Status               int      `json:"status"`
	PromptTokenCount     int      `json:"prompt_token_count"`
	CompletionTokenCount int      `json:"completion_token_count"`
	LatencyInMs          int      `json:"latency_in_ms"`
	Path                 string   `json:"path"`
	Method               string   `json:"method"`
	CustomId             string   `json:"custom_id"`
	Request              []byte   `json:"request"`
	Response             []byte   `json:"response"`
	UserId               string   `json:"userId"`
	Action               string   `json:"action"`
	PolicyId             string   `json:"policyId"`
}

type EventResponse struct {
	Events []*Event `json:"events"`
	Count  int      `json:"count"`
}

type EventRequest struct {
	UserIds         []string `json:"userIds"`
	CustomIds       []string `json:"customIds"`
	KeyIds          []string `json:"keyIds"`
	Tags            []string `json:"tags"`
	Start           int64    `json:"start"`
	End             int64    `json:"end"`
	Limit           int      `json:"limit"`
	Offset          int      `json:"offset"`
	RequestContent  string   `json:"requestContent"`
	ResponseContent string   `json:"responseContent"`
	PolicyIds       []string `json:"policyIds"`
	Actions         []string `json:"actions"`
	CostOrder       string   `json:"costOrder"`
	DateOrder       string   `json:"dateOrder"`
	ReturnCount     bool     `json:"returnCount"`
}

func (r *EventRequest) Validate() error {
	invalid := []string{}
	if r.Start == 0 {
		invalid = append(invalid, "start")
	}

	if r.End == 0 {
		invalid = append(invalid, "end")
	}

	for _, kid := range r.KeyIds {
		if len(kid) == 0 {
			invalid = append(invalid, "keyIds")
			break
		}
	}

	for _, uid := range r.UserIds {
		if len(uid) == 0 {
			invalid = append(invalid, "userIds")
			break
		}
	}

	for _, cid := range r.CustomIds {
		if len(cid) == 0 {
			invalid = append(invalid, "customIds")
			break
		}
	}

	for _, pid := range r.PolicyIds {
		if len(pid) == 0 {
			invalid = append(invalid, "policyIds")
			break
		}
	}

	for _, tag := range r.Tags {
		if len(tag) == 0 {
			invalid = append(invalid, "tags")
			break
		}
	}

	if len(invalid) > 0 {
		return internal_errors.NewValidationError(fmt.Sprintf("fields [%s] are invalid", strings.Join(invalid, ", ")))
	}

	if r.Start >= r.End {
		return internal_errors.NewValidationError(fmt.Sprintf("start %d cannot be larger than end %d", r.Start, r.End))
	}

	if len(r.CostOrder) != 0 && strings.ToLower(r.CostOrder) != "desc" && strings.ToLower(r.CostOrder) != "asc" {
		return internal_errors.NewValidationError(fmt.Sprintf("cost order is not valid %s", r.CostOrder))
	}

	if len(r.DateOrder) != 0 && strings.ToLower(r.DateOrder) != "desc" && strings.ToLower(r.DateOrder) != "asc" {
		return internal_errors.NewValidationError(fmt.Sprintf("date order is not valid %s", r.DateOrder))
	}

	if len(r.CostOrder) != 0 && len(r.DateOrder) != 0 {
		return internal_errors.NewValidationError("cost order and date order cannot be both present")
	}

	for _, a := range r.Actions {
		if a != "warned" && a != "allowed" && a != "blocked" {
			return internal_errors.NewValidationError(fmt.Sprintf("action cannot be %s", a))
		}
	}

	return nil
}
