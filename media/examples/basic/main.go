package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"mediakit"
)

func main() {

	client := mediakit.New(

		mediakit.WithFebboxCookie(os.Getenv("FEBBOX_UI_COOKIE")),
		mediakit.WithIntroDBKey(os.Getenv("INTRODB_API_KEY")),
		mediakit.WithIntroCache(true),

	)

	client.Warmup()

	hits, err := client.Search("breaking bad")

	if err != nil {

		log.Fatal(err)

	}

	if len(hits) == 0 {

		log.Fatal("no results")

	}

	show, err := client.ShowFromHit(hits[0])

	if err != nil {

		log.Fatal(err)

	}

	details, err := show.Details()

	if err != nil {

		log.Fatal(err)

	}

	fmt.Printf("Show: %s (%s)\n", details.Title, details.Year)

	ep := show.Episode(1, 1)

	title, _ := ep.Title()
	fmt.Printf("Episode: %s\n", title)

	if url, err := ep.StreamURL(720); err == nil {

		fmt.Printf("Stream URL: %s\n", url)

	} else {

		fmt.Printf("Stream URL unavailable: %v\n", err)

	}

	if intro, err := ep.Intro(); err == nil {

		if start, end, ok := intro.IntroWindow(); ok {

			fmt.Printf("Intro: %s → %s\n", start, end)

		}

	}

	if skipTo, err := ep.SkipIntroFrom(30 * time.Second); err == nil {

		fmt.Printf("Skip intro to: %s\n", skipTo)

	}

	catalog, err := client.LiveTV()

	if err != nil {

		log.Fatal(err)

	}

	for _, ch := range catalog.PopularUS(3) {

		info, _ := ch.Info()

		if stream, err := ch.Resolve(); err == nil {

			fmt.Printf("Live %s: %s\n", info.Name, stream.URL)

		}

	}

}
