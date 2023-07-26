package util

import (
	"regexp"
	"strings"
)

func GetVariableMap(str string) map[string]string {
	regexPattern := `({{\s*(.*?)\s*}})`
	regex := regexp.MustCompile(regexPattern)
	matches := regex.FindAllStringSubmatch(str, -1)

	variables := map[string]string{}
	for _, match := range matches {
		variables[match[1]] = strings.TrimSpace(match[2])
	}
	return variables
}
