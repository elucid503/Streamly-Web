package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"mediakit/internal/live/source"
)

func runCaptionWatch(interval, duration time.Duration, outDir string, channels []string) {

	h := newInspectHTTP(35 * time.Second)
	daddy := source.NewDaddyLive()

	if len(channels) == 0 {

		channels = []string{"SNY"}

	}
	deadline := time.Now().Add(duration)
	n := 0

	fmt.Println("DaddyLive CEA-608 mode watch")
	fmt.Println("GDELT rule: rollup ≈ program; popon/painton/none ≈ ad (US news)")
	fmt.Println()

	for time.Now().Before(deadline) || n == 0 {

		n++
		fmt.Printf("=== sample %02d %s ===\n", n, time.Now().Format("15:04:05"))

		for _, ch := range channels {

			sampleCaptions(h, daddy, ch, outDir, n)

		}

		fmt.Println()

		remain := time.Until(deadline)

		if remain <= 0 || n > 1 && remain < interval {

			break

		}

		wait := interval

		if wait > remain && n > 1 {

			wait = remain

		}

		time.Sleep(wait)

	}

	fmt.Printf("frames under %s\n", outDir)

}

func sampleCaptions(h *inspectHTTP, daddy source.Provider, channel, outDir string, n int) {

	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()

	stream, err := daddy.Resolve(ctx, source.Request{Name: channel, Country: "us"})

	if err != nil {

		fmt.Printf("%-5s resolve: %v\n", channel, err)
		return

	}

	body, status, err := h.getText(ctx, stream.URL, stream.Headers)

	if err != nil || status < 200 || status >= 300 || !strings.Contains(body, "#EXTM3U") {

		fmt.Printf("%-5s playlist fail status=%d err=%v\n", channel, status, err)
		return

	}

	mediaURL := stream.URL

	if isMasterPlaylist(body) {

		v, vErr := firstVariantURL(stream.URL, body)

		if vErr != nil {

			fmt.Printf("%-5s variant: %v\n", channel, vErr)
			return

		}

		mediaURL = v
		body, status, err = h.getText(ctx, mediaURL, stream.Headers)

		if err != nil || status < 200 || status >= 300 {

			fmt.Printf("%-5s media playlist fail: %v\n", channel, err)
			return

		}

	}

	segURL, err := lastSegmentURL(mediaURL, body)

	if err != nil {

		fmt.Printf("%-5s segment: %v\n", channel, err)
		return

	}

	seg, segStatus, err := h.get(ctx, segURL, stream.Headers, 2<<20)

	if err != nil || segStatus < 200 || segStatus >= 300 {

		fmt.Printf("%-5s segment get: status=%d err=%v\n", channel, segStatus, err)
		return

	}

	cc := extractCC608(seg)
	guess := "program"

	if cc.AdLikely() {

		guess = "AD?"

	}

	dir := filepath.Join(outDir, strings.ToLower(channel))
	_ = os.MkdirAll(dir, 0755)
	tsPath := filepath.Join(dir, fmt.Sprintf("%02d.ts", n))
	jpgPath := filepath.Join(dir, fmt.Sprintf("%02d.jpg", n))
	_ = os.WriteFile(tsPath, seg, 0644)

	ff := frameFromTS(ctx, tsPath, jpgPath)

	fmt.Printf("%-5s %-4s %s ffmpeg=%s\n", channel, guess, cc.String(), ff)

	_ = appendFile(filepath.Join(dir, "log.txt"),
		fmt.Sprintf("%s n=%02d guess=%s %s ffmpeg=%s\n",
			time.Now().Format(time.RFC3339), n, guess, cc.String(), ff))

}

func frameFromTS(ctx context.Context, tsPath, jpgPath string) string {

	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-nostdin", "-hide_banner", "-loglevel", "error",
		"-i", tsPath,
		"-an", "-frames:v", "1", "-q:v", "4",
		"-y", jpgPath,
	)

	out, err := cmd.CombinedOutput()

	if err != nil {

		msg := strings.TrimSpace(string(out))

		if msg == "" {

			msg = err.Error()

		}

		if len(msg) > 80 {

			msg = msg[:77] + "..."

		}

		return msg

	}

	st, statErr := os.Stat(jpgPath)

	if statErr != nil || st.Size() < 100 {

		return "no-frame"

	}

	return "ok"

}
