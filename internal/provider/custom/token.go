package custom

import (
	"github.com/pkoukk/tiktoken-go"
)

func NewTokenCounter() {
	// tiktoken.SetBpeLoader(tiktoken_loader.NewOfflineLoader())
}

func Count(input string) (int, error) {
	encoder, err := tiktoken.GetEncoding("cl100k_base")
	if err != nil {
		return 0, err
	}

	token := encoder.Encode(input, nil, nil)
	return len(token), nil
}
