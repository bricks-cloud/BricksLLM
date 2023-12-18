package azure

import (
	"strings"

	"github.com/pkoukk/tiktoken-go"
)

func Count(model string, input string) (int, error) {
	if strings.Contains(model, "ada") {
		encoder, err := tiktoken.GetEncoding("r50k_base")
		if err != nil {
			return 0, err
		}
		token := encoder.Encode(input, nil, nil)
		return len(token), nil
	}

	encoder, err := tiktoken.GetEncoding("cl100k_base")
	if err != nil {
		return 0, err
	}

	token := encoder.Encode(input, nil, nil)
	return len(token), nil
}
