package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"mediakit/internal/live/source"
)

type watchSample struct {

	Index int
	At time.Time
	Frame string
	SCTE string
	Tags []string
	Captions bool
	FFMPEG string

}

func runWatch(interval, duration time.Duration, outDir string) {

	httpClient := newInspectHTTP(35 * time.Second)

	targets := []target{

		{
			Name: "bloomberg-iptvorg",
			Kind: kindStreamly,
			Provider: source.NewIPTVOrg(),
			ChannelIDs: []string{"BloombergTV.us"},
			ChannelNames: []string{"Bloomberg"},
		},
		{
			Name: "xumo-fox-sports",
			Kind: kindXumo,
			XumoID: "99991196",
		},
		{
			Name: "daddylive-espn",
			Kind: kindStreamly,
			Provider: source.NewDaddyLive(),
			ChannelNames: []string{"ESPN"},
		},

	}

	deadline := time.Now().Add(duration)
	n := 0

	for time.Now().Before(deadline) || n == 0 {

		n++
		fmt.Printf("\n=== sample %02d %s ===\n", n, time.Now().Format("15:04:05"))

		var wg sync.WaitGroup

		for _, t := range targets {

			t := t
			wg.Add(1)

			go func() {

				defer wg.Done()
				sampleOne(httpClient, t, outDir, n, interval)

			}()

		}

		wg.Wait()

		if !time.Now().Add(interval).Before(deadline) && n > 1 {

			break

		}

		remain := time.Until(deadline)

		if remain <= 0 {

			break

		}

		wait := interval

		if wait > remain {

			wait = remain

		}

		time.Sleep(wait)

	}

	fmt.Printf("\nwrote frames under %s\n", outDir)

}

func sampleOne(h *inspectHTTP, t target, outDir string, n int, _ time.Duration) {

	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()

	stream, err := resolveTarget(ctx, h, t, "")

	if err == nil {

		stream.URL = preferWWW(stream.URL)

	}

	if err != nil {

		fmt.Printf("%-22s resolve: %v\n", t.Name, err)
		return

	}

	dir := filepath.Join(outDir, sanitizeName(t.Name))
	_ = os.MkdirAll(dir, 0755)
	frame := filepath.Join(dir, fmt.Sprintf("%02d.jpg", n))

	report, inspectErr := inspectHLS(ctx, h, stream.URL, stream.Headers)

	scte := "n/a"
	tags := []string{}
	caps := false

	if inspectErr == nil {

		tags = append(report.MasterTags, report.MediaTags...)
		caps = report.ClosedCaptions || hasCaptionSegment(report.SegmentHits)

		if report.SegmentURL != "" {

			seg, status, segErr := h.get(ctx, report.SegmentURL, stream.Headers, segmentLimit)

			if segErr == nil && status >= 200 && status < 300 {

				msgs := parseSCTEMessages(seg)
				scte = summarizeSCTE(msgs)

				if len(msgs) == 0 && hasStrongSegment(report.SegmentHits) {

					scte = strings.Join(report.SegmentHits, ",")

				}

				if hasCaptionSegment(report.SegmentHits) {

					caps = true

				}

			}

		}

	} else {

		scte = "inspect: " + inspectErr.Error()

	}

	ffErr := grabFrame(ctx, stream.URL, stream.Headers, frame)

	ff := "ok"

	if ffErr != nil {

		ff = ffErr.Error()

	}

	tagStr := strings.Join(uniq(tags), ",")

	if tagStr == "" {

		tagStr = "-"

	}

	fmt.Printf("%-22s scte=%-18s cc=%-5v ffmpeg=%s tags=%s\n",
		t.Name, scte, caps, shortErr(ff), tagStr)

	_ = appendFile(filepath.Join(dir, "log.txt"),
		fmt.Sprintf("%s sample=%02d scte=%s cc=%v ffmpeg=%s tags=%s url=%s\n",
			time.Now().Format(time.RFC3339), n, scte, caps, ff, tagStr, stream.URL))

}

func grabFrame(ctx context.Context, streamURL string, headers map[string]string, out string) error {

	args := []string{
		"-nostdin",
		"-hide_banner",
		"-loglevel", "error",
		"-user_agent", inspectUA,
		"-rw_timeout", "15000000",
	}

	if ref := headers["Referer"]; ref != "" {

		hdr := "Referer: " + ref + "\r\n"

		if o := headers["Origin"]; o != "" {

			hdr += "Origin: " + o + "\r\n"

		}

		args = append(args, "-headers", hdr)

	}

	args = append(args,
		"-i", streamURL,
		"-an",
		"-frames:v", "1",
		"-q:v", "4",
		"-y",
		out,
	)

	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	stderr, err := cmd.CombinedOutput()

	if err != nil {

		msg := strings.TrimSpace(string(stderr))

		if msg == "" {

			msg = err.Error()

		}

		if len(msg) > 180 {

			msg = msg[:180]

		}

		return fmt.Errorf("%s", msg)

	}

	if st, statErr := os.Stat(out); statErr != nil || st.Size() < 100 {

		return fmt.Errorf("no frame written")

	}

	return nil

}

func preferWWW(raw string) string {

	return strings.Replace(raw, "://bloomberg.com/", "://www.bloomberg.com/", 1)

}

func sanitizeName(s string) string {

	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "-")

	return s

}

func uniq(in []string) []string {

	seen := map[string]bool{}
	var out []string

	for _, s := range in {

		if s == "" || seen[s] {

			continue

		}

		seen[s] = true
		out = append(out, s)

	}

	return out

}

func shortErr(s string) string {

	s = strings.ReplaceAll(s, "\n", " ")

	if len(s) > 40 {

		return s[:37] + "..."

	}

	return s

}

func appendFile(path, line string) error {

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)

	if err != nil {

		return err

	}

	defer f.Close()
	_, err = f.WriteString(line)
	return err

}
