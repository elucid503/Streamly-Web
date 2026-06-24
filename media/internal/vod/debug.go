package vod

import (
	"log"
	"os"
	"strings"
)

func streamDebugf(format string, args ...any) {

	switch strings.ToLower(strings.TrimSpace(os.Getenv("STREAM_DEBUG"))) {

	case "1", "true", "yes", "on":

		log.Printf("[stream-debug] "+format, args...)

	default:

	}

}