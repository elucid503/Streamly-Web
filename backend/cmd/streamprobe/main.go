package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"streamly/internal/config"
	"streamly/internal/services"
)

func main() {

	movieID := flag.Int("movie", 0, "showbox movie id")
	showID := flag.Int("show", 0, "showbox show id")
	season := flag.Int("season", 0, "season number")
	episode := flag.Int("episode", 0, "episode number")

	flag.Parse()

	os.Setenv("STREAM_DEBUG", "1")

	cfg, err := config.Load()

	if err != nil {

		log.Fatal(err)

	}

	fmt.Println("--- config ---")
	fmt.Printf("VIXSRC_SERVER=%v\n", cfg.VixsrcServerEnabled)
	fmt.Printf("VIXSRC_PROXY_URL=%s\n", maskProxyURL(cfg.VixsrcProxyURL))
	fmt.Printf("FEBBOX_UI_COOKIE=%t\n", cfg.FebboxCookie != "")
	fmt.Printf("TMDB_API_KEY=%t\n", cfg.TMDBAPIKey != "")
	fmt.Println()

	media := services.NewMediaService(cfg)

	if *movieID > 0 {

		probeMovie(media, *movieID)

		return

	}

	if *showID > 0 && *season > 0 && *episode > 0 {

		probeEpisode(media, *showID, *season, *episode)

		return

	}

	log.Fatal("usage: streamprobe -movie 4059  OR  streamprobe -show 165 -season 2 -episode 2")

}

func probeMovie(media *services.MediaService, id int) {

	fmt.Printf("=== movie %d ===\n", id)

	details, err := media.MovieDetails(id)

	if err != nil {

		fmt.Printf("details error: %v\n", err)

	} else {

		fmt.Printf("title=%q year=%s\n", details.Title, details.Year)

	}

	qualities, err := media.MovieQualities(id)

	fmt.Printf("qualities err=%v count=%d\n", err, len(qualities))

	for i, q := range qualities {

		if i >= 3 {

			break

		}

		fmt.Printf("  [%d] %s h=%d hls=%t url=%s\n", i, q.Label, q.Height, q.IsHLS, q.URL)

	}

	dto := services.BuildStreamDTO(qualities)

	if dto == nil {

		fmt.Println("BuildStreamDTO: nil (would return 404)")

		return

	}

	fmt.Printf("BuildStreamDTO: %d playable qualities\n", len(dto.Qualities))

}

func probeEpisode(media *services.MediaService, showID, season, episode int) {

	fmt.Printf("=== show %d S%02dE%02d ===\n", showID, season, episode)

	details, err := media.ShowDetails(showID)

	if err != nil {

		fmt.Printf("details error: %v\n", err)

	} else {

		fmt.Printf("title=%q year=%s\n", details.Title, details.Year)

	}

	qualities, err := media.EpisodeQualities(showID, season, episode)

	fmt.Printf("qualities err=%v count=%d\n", err, len(qualities))

	for i, q := range qualities {

		if i >= 3 {

			break

		}

		fmt.Printf("  [%d] %s h=%d hls=%t url=%s\n", i, q.Label, q.Height, q.IsHLS, q.URL)

	}

	dto := services.BuildStreamDTO(qualities)

	if dto == nil {

		fmt.Println("BuildStreamDTO: nil (would return 404)")

		return

	}

	fmt.Printf("BuildStreamDTO: %d playable qualities\n", len(dto.Qualities))

}

func maskProxyURL(raw string) string {

	raw = strings.TrimSpace(raw)

	if raw == "" {

		return "(not set)"

	}

	if !strings.Contains(raw, "://") {

		raw = "http://" + raw

	}

	if at := strings.LastIndex(raw, "@"); at >= 0 {

		schemeEnd := strings.Index(raw, "://")

		if schemeEnd >= 0 {

			return raw[:schemeEnd+3] + "***@" + raw[at+1:]

		}

	}

	return raw

}