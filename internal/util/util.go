package util

import (
	"net/http"
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

func FilterHeaders(headers http.Header, filters []string) map[string][]string {
	result := map[string][]string{}
	for header, val := range headers {

		filtered := false
		for _, filter := range filters {
			if strings.ToLower(filter) == strings.ToLower(header) {
				filtered = true
				break
			}
		}

		if filtered {
			continue
		}

		result[header] = val
	}

	return result
}
