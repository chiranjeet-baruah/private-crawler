package types

import "strings"

func normalizeErrors(message string) (code string) {
	if strings.Contains(message, "the request is invalid") {
		return "DOES_NOT_EXIST"
	}
	return "AMAZON_NORMALIZE_ERROR"
}
