package intro

import (
	"time"

	"mediakit/internal/introdb"
)

type config struct {

	durationMs *int64

}

// Option configures a TheIntroDB lookup.
type Option func(*config)

// WithDuration hints the media runtime in milliseconds for duration-aware intro matching.
func WithDuration(duration time.Duration) Option {

	return func(c *config) {

		ms := duration.Milliseconds()
		c.durationMs = &ms

	}

}

// Segment is a community-verified intro/recap/credits window.
type Segment struct {

	Start time.Duration
	End *time.Duration

}

// Data is normalized intro timing from TheIntroDB.
type Data struct {

	TMDBId int
	Type string
	Segments []Segment

}

// IntroWindow returns the first viable intro segment window.
func (d *Data) IntroWindow() (start, end time.Duration, ok bool) {

	startMs, endMs, found := introdb.IntroWindow(ToRecord(d))

	if !found {

		return 0, 0, false

	}

	return time.Duration(startMs) * time.Millisecond, time.Duration(endMs) * time.Millisecond, true

}

// BuildQuery constructs a TheIntroDB media query from title identifiers.
func BuildQuery(tmdbID int, imdbID string, season, episode int, durationMs *int64) (introdb.MediaQuery, error) {

	return introdb.QueryForTitle(tmdbID, imdbID, season, episode, durationMs)

}

// ApplyOptions applies Option values and returns the resolved config.
func ApplyOptions(opts []Option) *config {

	cfg := &config{}

	for _, opt := range opts {

		opt(cfg)

	}

	return cfg

}

// DurationMs returns the optional duration hint from a config.
func DurationMs(cfg *config) *int64 {

	return cfg.durationMs

}

// FromRecord converts an introdb.MediaRecord to Data.
func FromRecord(record *introdb.MediaRecord) *Data {

	if record == nil {

		return nil

	}

	data := &Data{

		TMDBId: record.TMDBId,
		Type: record.Type,

	}

	for _, segment := range record.Intro {

		start := time.Duration(segment.StartMs) * time.Millisecond

		var end *time.Duration

		if segment.EndMs != nil {

			value := time.Duration(*segment.EndMs) * time.Millisecond
			end = &value

		}

		data.Segments = append(data.Segments, Segment{Start: start, End: end})

	}

	return data

}

// ToRecord converts Data back to an introdb.MediaRecord for skip/credits calculations.
func ToRecord(data *Data) *introdb.MediaRecord {

	if data == nil {

		return nil

	}

	record := &introdb.MediaRecord{

		TMDBId: data.TMDBId,
		Type: data.Type,

	}

	for _, segment := range data.Segments {

		startMs := segment.Start.Milliseconds()

		var endMs *int64

		if segment.End != nil {

			value := segment.End.Milliseconds()
			endMs = &value

		}

		record.Intro = append(record.Intro, introdb.SegmentTimestamp{

			StartMs: startMs,
			EndMs: endMs,

		})

	}

	return record

}
