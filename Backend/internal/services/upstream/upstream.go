package upstream

import (
	"strings"
	"time"
)

// Retry calls fn up to attempts times, retrying only on rate-limit errors.
func Retry[T any](attempts int, fn func() (T, error)) (T, error) {

	var zero T

	var last error

	backoff := 5 * time.Second

	for i := 0; i < attempts; i++ {

		if i > 0 {

			time.Sleep(backoff)
			backoff *= 2

		}

		result, err := fn()

		if err == nil {

			return result, nil

		}

		last = err

		if !IsRateLimitError(err) {

			return zero, err

		}

	}

	return zero, last

}

// IsRateLimitError reports whether an error looks like an upstream rate limit.
func IsRateLimitError(err error) bool {

	if err == nil {

		return false

	}

	msg := strings.ToLower(err.Error())

	return strings.Contains(msg, "429") ||
		strings.Contains(msg, "rate limit") ||
		strings.Contains(msg, "1015") ||
		strings.Contains(msg, "febbox:") ||
		strings.Contains(msg, "525")

}
