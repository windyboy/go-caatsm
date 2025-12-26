package aviation

import (
	"context"
	"regexp"
	"time"
)

const (
	// DefaultRegexTimeout is the maximum time allowed for regex matching operations.
	// This prevents ReDoS (Regular Expression Denial of Service) attacks from
	// maliciously crafted inputs that cause catastrophic backtracking.
	DefaultRegexTimeout = 100 * time.Millisecond
)

// MatchWithTimeout executes a regex match with timeout protection.
// It runs the regex matching in a goroutine and returns an error if the
// operation exceeds the specified timeout duration.
//
// This is critical for preventing ReDoS attacks where complex patterns
// (especially the FPL pattern with nested quantifiers) could hang indefinitely
// on malicious input.
//
// Parameters:
//   - re: The compiled regular expression to match
//   - input: The input string to match against
//   - timeout: Maximum duration allowed for the match operation
//
// Returns:
//   - []string: The match result (same format as regexp.FindStringSubmatch)
//   - error: ValidationError if timeout occurs, nil otherwise
func MatchWithTimeout(re *regexp.Regexp, input string, timeout time.Duration) ([]string, error) {
	type result struct {
		match []string
	}

	resultChan := make(chan result, 1)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Run regex matching in a goroutine
	go func() {
		match := re.FindStringSubmatch(input)
		resultChan <- result{match: match}
	}()

	// Wait for either result or timeout
	select {
	case res := <-resultChan:
		return res.match, nil
	case <-ctx.Done():
		return nil, &ValidationError{
			Field:   "regex_timeout",
			Message: "regex matching exceeded timeout",
		}
	}
}
