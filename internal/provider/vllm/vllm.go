package vllm

import goopenai "github.com/sashabaranov/go-openai"

type CompletionRequest struct {
	goopenai.CompletionRequest
	UseBeamSearch              bool    `json:"use_beam_search,omitempty"`
	TopK                       int     `json:"top_k,omitempty"`
	MinP                       int     `json:"min_p,omitempty"`
	RepetitionPenalty          float64 `json:"repetition_penalty,omitempty"`
	LengthPenalty              float64 `json:"length_penalty,omitempty"`
	EarlyStopping              bool    `json:"early_stopping,omitempty"`
	StopTokenIds               []int   `json:"stop_token_ids,omitempty"`
	IgnoreEos                  bool    `json:"ignore_eos,omitempty"`
	MinTokens                  int     `json:"min_tokens,omitempty"`
	SkipSpecialTokens          bool    `json:"skip_special_tokens,omitempty"`
	SpacesBetweenSpecialTokens bool    `json:"spaces_between_special_tokens,omitempty"`
}

type ChatRequest struct {
	goopenai.ChatCompletionRequest
	BestOf                     int     `json:"best_of,omitempty"`
	UseBeamSearch              bool    `json:"use_beam_search,omitempty"`
	TopK                       int     `json:"top_k,omitempty"`
	MinP                       int     `json:"min_p,omitempty"`
	RepetitionPenalty          float64 `json:"repetition_penalty,omitempty"`
	LengthPenalty              float64 `json:"length_penalty,omitempty"`
	EarlyStopping              bool    `json:"early_stopping,omitempty"`
	IgnoreEos                  bool    `json:"ignore_eos,omitempty"`
	MinTokens                  int     `json:"min_tokens,omitempty"`
	StopTokenIds               []int   `json:"stop_token_ids,omitempty"`
	SkipSpecialTokens          bool    `json:"skip_special_tokens,omitempty"`
	SpacesBetweenSpecialTokens bool    `json:"spaces_between_special_tokens,omitempty"`
}
