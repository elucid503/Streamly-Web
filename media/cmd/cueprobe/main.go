package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type probeResult struct {

	Name string `json:"name"`
	FMHY string `json:"fmhy"`
	Kind string `json:"kind"`
	Status string `json:"status"`
	Error string `json:"error,omitempty"`
	Detail string `json:"detail,omitempty"`
	StreamURL string `json:"stream_url,omitempty"`
	MasterTags []string `json:"master_tags,omitempty"`
	MediaTags []string `json:"media_tags,omitempty"`
	SegmentHits []string `json:"segment_hits,omitempty"`
	ClosedCaptions bool `json:"closed_captions,omitempty"`
	StrongCue bool `json:"strong_cue"`
	Hint bool `json:"hint"`

}

func main() {

	channel := flag.String("channel", "ESPN", "channel name (comma-separated for -captions)")
	timeout := flag.Duration("timeout", 40*time.Second, "per-target timeout")
	workers := flag.Int("workers", 4, "parallel probes")
	jsonOut := flag.Bool("json", false, "print JSON instead of a table")
	watch := flag.Bool("watch", false, "sample working sources with ffmpeg frames")
	captions := flag.Bool("captions", false, "sample DaddyLive CEA-608 mode vs ffmpeg frames")
	interval := flag.Duration("interval", 60*time.Second, "watch sample interval")
	duration := flag.Duration("duration", 6*time.Minute, "watch duration")
	outDir := flag.String("out", "", "watch output directory")
	flag.Parse()

	if *captions {

		dir := *outDir

		if dir == "" {

			dir = filepath.Join(os.TempDir(), "ccwatch")

		}

		if *interval == 60*time.Second {

			*interval = 20 * time.Second

		}

		runCaptionWatch(*interval, *duration, dir, splitChannels(*channel))
		return

	}

	if *watch {

		dir := *outDir

		if dir == "" {

			dir = filepath.Join(os.TempDir(), "cuewatch")

		}

		runWatch(*interval, *duration, dir)
		return

	}

	targets := fmhyLiveTVTargets()
	httpClient := newInspectHTTP(*timeout)

	results := make([]probeResult, len(targets))
	sem := make(chan struct{}, *workers)
	var wg sync.WaitGroup

	for i, t := range targets {

		i := i
		t := t
		wg.Add(1)

		go func() {

			defer wg.Done()

			sem <- struct{}{}
			defer func() { <-sem }()

			results[i] = probeOne(httpClient, t, *channel, *timeout)

		}()

	}

	wg.Wait()

	if *jsonOut {

		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(results)
		return

	}

	printTable(results)

}

func splitChannels(raw string) []string {

	var out []string

	for _, p := range strings.Split(raw, ",") {

		p = strings.TrimSpace(p)

		if p != "" {

			out = append(out, p)

		}

	}

	return out

}

func probeOne(h *inspectHTTP, t target, channel string, timeout time.Duration) probeResult {

	out := probeResult{

		Name: t.Name,
		FMHY: t.FMHY,
		Kind: string(t.Kind),

	}

	if t.Kind == kindSkip {

		out.Status = "skip"
		out.Error = t.Note
		return out

	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	stream, err := resolveTarget(ctx, h, t, channel)

	if err != nil {

		out.Status = "no-stream"
		out.Error = err.Error()
		return out

	}

	out.StreamURL = stream.URL
	out.Detail = stream.Detail

	report, inspectErr := inspectHLS(ctx, h, stream.URL, stream.Headers)

	if inspectErr != nil {

		out.Status = "unreadable"
		out.Error = inspectErr.Error()
		return out

	}

	out.MasterTags = report.MasterTags
	out.MediaTags = report.MediaTags
	out.SegmentHits = report.SegmentHits
	out.ClosedCaptions = report.ClosedCaptions || hasCaptionSegment(report.SegmentHits)
	out.StrongCue = report.anyCue()
	out.Hint = report.anyHint()

	if out.StrongCue {

		out.Status = "cues"

	} else if out.Hint || out.ClosedCaptions {

		out.Status = "hints"

	} else {

		out.Status = "clean"

	}

	return out

}

func printTable(results []probeResult) {

	fmt.Printf("%-28s %-10s %-8s %s\n", "SOURCE", "STATUS", "CUES", "DETAIL")
	fmt.Println(strings.Repeat("-", 110))

	var cues, hints, clean, nostream, skipped int

	for _, r := range results {

		switch r.Status {

		case "cues":
			cues++
		case "hints":
			hints++
		case "clean":
			clean++
		case "skip":
			skipped++
		default:
			nostream++

		}

		mark := "-"

		if r.StrongCue {

			mark = "YES"

		} else if r.Hint || r.ClosedCaptions {

			mark = "hint"

		}

		detail := r.Error

		if detail == "" {

			parts := append([]string{}, r.MediaTags...)
			parts = append(parts, r.SegmentHits...)

			if r.ClosedCaptions {

				parts = append(parts, "CC")

			}

			if r.Detail != "" {

				parts = append([]string{r.Detail}, parts...)

			}

			detail = strings.Join(parts, ", ")

		}

		if len(detail) > 70 {

			detail = detail[:67] + "..."

		}

		fmt.Printf("%-28s %-10s %-8s %s\n", r.Name, r.Status, mark, detail)

	}

	fmt.Println(strings.Repeat("-", 110))
	fmt.Printf("cues=%d  hints=%d  clean=%d  no-stream=%d  skip=%d\n",
		cues, hints, clean, nostream, skipped)
	fmt.Println()
	fmt.Println("cues  = HLS CUE/SCTE tags or MPEG-TS PMT stream_type 0x86 (real splice PID)")
	fmt.Println("hints = discontinuity, PDT, DATERANGE, CC1/GA94 captions, or ASSET beacons")
	fmt.Println("clean = playable HLS with none of the above")

}
