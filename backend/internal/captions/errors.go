package captions

import "errors"

var (

	ErrNoSubtitle  = errors.New("captions: no English subtitles found")
	ErrUnconfigured = errors.New("captions: subtitle provider not configured")
	ErrUnauthorized = errors.New("captions: subtitle provider unauthorized")
	ErrRateLimited  = errors.New("captions: subtitle provider rate limited")

)
