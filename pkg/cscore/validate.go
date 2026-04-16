package cscore

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

const (
	maxTopicLen    = 128
	maxQueryLen    = 512
	maxExprLen     = 1024
	maxCIDRLen     = 128
	maxCategoryLen = 128
	maxMarkdownLen = 1 << 20 // 1 MB
)

type validationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e *validationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

func validateTopic(name string) error {
	if name == "" {
		return &validationError{"topic", "empty topic name"}
	}
	if len(name) > maxTopicLen {
		return &validationError{"topic", "exceeds maximum length"}
	}
	if !utf8.ValidString(name) {
		return &validationError{"topic", "invalid UTF-8"}
	}
	if strings.Contains(name, "..") || strings.ContainsAny(name, "/\\") {
		return &validationError{"topic", "contains path separator or traversal"}
	}
	if strings.ContainsRune(name, 0) {
		return &validationError{"topic", "contains null byte"}
	}
	return nil
}

func validateQuery(q string) error {
	if len(q) > maxQueryLen {
		return &validationError{"query", "exceeds maximum length"}
	}
	if !utf8.ValidString(q) {
		return &validationError{"query", "invalid UTF-8"}
	}
	if strings.ContainsRune(q, 0) {
		return &validationError{"query", "contains null byte"}
	}
	return nil
}

func validateExpr(expr string) error {
	if expr == "" {
		return &validationError{"expr", "empty expression"}
	}
	if len(expr) > maxExprLen {
		return &validationError{"expr", "exceeds maximum length"}
	}
	if strings.ContainsRune(expr, 0) {
		return &validationError{"expr", "contains null byte"}
	}
	return nil
}

func validateCIDR(input string) error {
	if input == "" {
		return &validationError{"cidr", "empty input"}
	}
	if len(input) > maxCIDRLen {
		return &validationError{"cidr", "exceeds maximum length"}
	}
	return nil
}

func validateCategory(name string) error {
	if name == "" {
		return &validationError{"category", "empty category"}
	}
	if len(name) > maxCategoryLen {
		return &validationError{"category", "exceeds maximum length"}
	}
	if strings.Contains(name, "..") || strings.ContainsAny(name, "/\\") {
		return &validationError{"category", "contains path separator or traversal"}
	}
	return nil
}

func validateMarkdown(md string) error {
	if md == "" {
		return &validationError{"markdown", "empty input"}
	}
	if len(md) > maxMarkdownLen {
		return &validationError{"markdown", "exceeds 1MB limit"}
	}
	return nil
}
