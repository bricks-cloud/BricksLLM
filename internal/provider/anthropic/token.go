package anthropic

import (
	"encoding/base64"
	"encoding/json"
	"path"
	"strings"

	"github.com/bricks-cloud/bricksllm/internal/provider/anthropic/assets"
	"github.com/pkoukk/tiktoken-go"
)

type TokenCounter struct {
	encoding        *tiktoken.Encoding
	core            *tiktoken.CoreBPE
	specialTokenSet map[string]any
}

type anthropicConfigurations struct {
	ExplicitNVocab int            `json:"explicit_n_vocab"`
	Pattern        string         `json:"pat_str"`
	SpecialTokens  map[string]int `json:"special_tokens"`
	BpeRanks       string         `json:"bpe_ranks"`
}

func NewTokenCounter() (*TokenCounter, error) {
	ac, err := loadAnthropicConfig("claude.json")
	if err != nil {
		return nil, err
	}

	bpeRanks := make(map[string]int)
	lines := strings.Split(ac.BpeRanks, " ")
	rank := 1
	for _, line := range lines {
		token, err := base64.StdEncoding.DecodeString(line)
		if err != nil {
			continue
		}

		bpeRanks[string(token)] = rank
		rank++
	}

	bpe, err := tiktoken.NewCoreBPE(bpeRanks, ac.SpecialTokens, ac.Pattern)
	if err != nil {
		return nil, err
	}

	specialTokensSet := map[string]any{}
	for k := range ac.SpecialTokens {
		specialTokensSet[k] = true
	}

	return &TokenCounter{
		encoding: &tiktoken.Encoding{
			Name:           "anthropic",
			PatStr:         ac.Pattern,
			MergeableRanks: bpeRanks,
			SpecialTokens:  ac.SpecialTokens,
			ExplicitNVocab: ac.ExplicitNVocab,
		},
		core:            bpe,
		specialTokenSet: specialTokensSet,
	}, nil
}

func loadAnthropicConfig(configFile string) (*anthropicConfigurations, error) {
	baseFileName := path.Base(configFile)
	contents, err := assets.Assets.ReadFile(baseFileName)
	if err != nil {
		return nil, err
	}

	configs := &anthropicConfigurations{}
	err = json.Unmarshal(contents, configs)
	if err != nil {
		return nil, err
	}

	return configs, nil
}

func (tc *TokenCounter) Count(input string) int {
	tt := tiktoken.NewTiktoken(tc.core, tc.encoding, tc.specialTokenSet)
	token := tt.Encode(input, nil, nil)
	return len(token)
}
