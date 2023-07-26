package prompt

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/bricks-cloud/atlas/config"
)

type Prompt struct {
	template string
}

func Replace(value map[string]interface{}, input map[string]config.InputValue, template string) (string, error) {
	re, err := regexp.Compile(`\{\{.*\}\}`)
	if err != nil {
		return "", fmt.Errorf("error when compling regex: %v", err)
	}

	replacements := map[string]string{}
	matched := re.FindAllString(template, -1)

	for _, v := range matched {
		replacements[v] = ""
	}

	result := template
	for v, replacement := range replacements {
		result = strings.ReplaceAll(result, v, replacement)
	}

	return result, nil
}
