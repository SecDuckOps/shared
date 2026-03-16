package validation

import (
	"strings"

	"github.com/SecDuckOps/shared/types"
)

// trims whitespace and returns an error if the result is empty.
func RequireNonEmpty(value, fieldName string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", types.Newf(types.ErrCodeInvalidInput, "%s is required", fieldName)
	}
	return value, nil
}

// trims, checks empty, checks max length
func RequireMaxLen(value, fieldName string, maxLen int) (string, error) {
	value, err := RequireNonEmpty(value, fieldName)
	if err != nil {
		return "", err
	}
	if len(value) > maxLen {
		return "", types.Newf(types.ErrCodeInvalidInput, "%s exceeds max length of %d", fieldName, maxLen)
	}
	return value, nil
}

// identifier fields (IDs) with a standard max of 256
func RequireIdentifier(value, fieldName string) (string, error) {
	return RequireMaxLen(value, fieldName, 256)
}
