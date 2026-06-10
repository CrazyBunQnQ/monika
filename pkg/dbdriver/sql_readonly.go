package dbdriver

import (
	"fmt"
	"regexp"
	"strings"
)

var allowedSQLPattern = regexp.MustCompile(`(?i)^\s*(SELECT|SHOW|DESCRIBE|DESC|EXPLAIN|PRAGMA)\s`)

func ValidateReadOnlySQL(query string) error {
	q := strings.TrimSpace(query)
	if q == "" {
		return fmt.Errorf("empty query")
	}
	if !allowedSQLPattern.MatchString(q) {
		return fmt.Errorf("query rejected: only SELECT, SHOW, DESCRIBE, EXPLAIN, and PRAGMA statements are allowed")
	}
	return nil
}
