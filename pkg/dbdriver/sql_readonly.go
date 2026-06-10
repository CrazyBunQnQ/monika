package dbdriver

import (
	"fmt"
	"strings"
	"unicode"
)

var allowedPrefixes = []string{
	"SELECT ",
	"SHOW ",
	"DESCRIBE ",
	"DESC ",
	"EXPLAIN ",
	"PRAGMA ",
}

var forbiddenSubstrings = []string{
	"INTO OUTFILE",
	"INTO DUMPFILE",
	"FOR UPDATE",
	"FOR SHARE",
	"INTO @",
	"INTO @@",
}

func ValidateReadOnlySQL(query string) error {
	q := strings.TrimSpace(query)
	if q == "" {
		return fmt.Errorf("empty query")
	}

	if strings.Contains(q, ";") {
		return fmt.Errorf("query rejected: multi-statement queries are not allowed")
	}

	upper := strings.ToUpper(q)
	for _, forbidden := range forbiddenSubstrings {
		if strings.Contains(upper, forbidden) {
			return fmt.Errorf("query rejected: write operation detected (%s)", forbidden)
		}
	}

	normalized := strings.ToUpper(string(q[0])) + q[1:]
	normalizedUpper := strings.ToUpper(normalized)

	if strings.HasPrefix(normalizedUpper, "PRAGMA ") {
		if strings.Contains(normalizedUpper, "=") {
			return fmt.Errorf("query rejected: PRAGMA write operations are not allowed")
		}
		return nil
	}

	if strings.HasPrefix(normalizedUpper, "WITH RECURSIVE ") || strings.HasPrefix(normalizedUpper, "WITH ") {
		mainStmt := extractCteMainStatement(normalizedUpper)
		if mainStmt == "" {
			return fmt.Errorf("query rejected: unable to parse CTE structure")
		}
		if !isAllowedPrefix(mainStmt) {
			return fmt.Errorf("query rejected: CTE with %s main statement is not allowed", firstWord(mainStmt))
		}
		return nil
	}

	if !isAllowedPrefix(normalizedUpper) {
		return fmt.Errorf("query rejected: only SELECT, SHOW, DESCRIBE, EXPLAIN, and PRAGMA statements are allowed (got %s)", firstWord(normalizedUpper))
	}

	return nil
}

func isAllowedPrefix(upper string) bool {
	for _, prefix := range allowedPrefixes {
		if strings.HasPrefix(upper, prefix) {
			return true
		}
	}
	return false
}

func firstWord(s string) string {
	s = strings.TrimSpace(s)
	i := strings.IndexFunc(s, unicode.IsSpace)
	if i < 0 {
		return s
	}
	return s[:i]
}

func extractCteMainStatement(upper string) string {
	depth := 0
	inString := false
	stringChar := byte(0)

	for i := 0; i < len(upper); i++ {
		ch := upper[i]

		if inString {
			if ch == stringChar {
				inString = false
			}
			continue
		}

		if ch == '\'' {
			inString = true
			stringChar = '\''
			continue
		}

		if ch == '(' {
			depth++
		} else if ch == ')' {
			depth--
			if depth == 0 {
				rest := strings.TrimSpace(upper[i+1:])
				if isAllowedPrefix(rest) || strings.HasPrefix(rest, "INSERT") || strings.HasPrefix(rest, "UPDATE") || strings.HasPrefix(rest, "DELETE") {
					return rest
				}
			}
		}
	}

	rest := upper
	for {
		rest = strings.TrimSpace(rest)
		if !strings.HasPrefix(rest, "WITH RECURSIVE ") && !strings.HasPrefix(rest, "WITH ") {
			break
		}
		asIdx := strings.Index(rest, " AS (")
		if asIdx < 0 {
			break
		}
		depth2 := 0
		start := asIdx + 4
		found := false
		for j := start; j < len(rest); j++ {
			if rest[j] == '(' {
				depth2++
			} else if rest[j] == ')' {
				depth2--
				if depth2 == 0 {
					rest = strings.TrimSpace(rest[j+1:])
					found = true
					break
				}
			}
		}
		if !found {
			break
		}
	}

	return rest
}
