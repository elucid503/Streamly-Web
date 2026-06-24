package providers

import (
	"log"
	"os"
	"strings"
)

func streamDebugEnabled() bool {

	switch strings.ToLower(strings.TrimSpace(os.Getenv("STREAM_DEBUG"))) {

	case "1", "true", "yes", "on":

		return true

	default:

		return false

	}

}

func streamDebugf(format string, args ...any) {

	if !streamDebugEnabled() {

		return

	}

	log.Printf("[stream-debug] "+format, args...)

}