package mediakit

import (
	"time"

	"mediakit/internal/introdb"
)

type introConfig struct {
	durationMs *int64
}

// IntroOption configures TheIntroDB lookups.
type IntroOption func(*introConfig)

// WithDuration hints the runtime in milliseconds for duration-aware intro matching.
func WithDuration(duration time.Duration) IntroOption {
	return func(c *introConfig) {
		ms := duration.Milliseconds()
		c.durationMs = &ms
	}
}

func toIntroData(record *introdb.MediaRecord) *IntroData {
	if record == nil {
		return nil
	}

	data := &IntroData{
		TMDBId: record.TMDBId,
		Type:   record.Type,
	}

	for _, segment := range record.Intro {
		start := time.Duration(segment.StartMs) * time.Millisecond
		var end *time.Duration
		if segment.EndMs != nil {
			value := time.Duration(*segment.EndMs) * time.Millisecond
			end = &value
		}
		data.Segments = append(data.Segments, IntroSegment{Start: start, End: end})
	}

	return data
}

func toMediaRecord(data *IntroData) *introdb.MediaRecord {
	if data == nil {
		return nil
	}

	record := &introdb.MediaRecord{
		TMDBId: data.TMDBId,
		Type:   data.Type,
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
			EndMs:   endMs,
		})
	}

	return record
}

// IntroWindow returns the first viable intro segment window.
func (d *IntroData) IntroWindow() (start, end time.Duration, ok bool) {
	startMs, endMs, found := introdb.IntroWindow(toMediaRecord(d))
	if !found {
		return 0, 0, false
	}
	return time.Duration(startMs) * time.Millisecond, time.Duration(endMs) * time.Millisecond, true
}