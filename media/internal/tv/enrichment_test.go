package tv

import "testing"

func TestChannelMetadataMatchPrefersCountryHint(t *testing.T) {

	index := buildChannelMetadataIndex([]iptvChannel{

		{

			ID:         "ESPN.au",
			Name:       "ESPN",
			Country:    "AU",
			Categories: []string{"sports"},
		},
		{

			ID:         "ESPN.us",
			Name:       "ESPN",
			Country:    "US",
			Categories: []string{"sports"},
		},
	}, []iptvLogo{

		{

			Channel: "ESPN.us",
			InUse:   true,
			Width:   960,
			Height:  238,
			Format:  "PNG",
			URL:     "https://example.test/espn.png",
		},
	})

	match, ok := index.match("espn usa")

	if !ok {

		t.Fatal("expected ESPN USA to match metadata")

	}

	if match.ID != "ESPN.us" {

		t.Fatalf("expected ESPN.us, got %s", match.ID)

	}

	if match.Logo == "" {

		t.Fatal("expected matched metadata to carry the best logo")

	}

}

func TestEnrichChannelCatalogUsesAliasesAndRegionFallback(t *testing.T) {

	index := buildChannelMetadataIndex([]iptvChannel{

		{

			ID:         "ABC.us",
			Name:       "ABC",
			Country:    "US",
			Categories: []string{"general"},
		},
		{

			ID:         "AmericanHeroesChannel.us",
			Name:       "American Heroes Channel",
			AltNames:   []string{"AHC"},
			Country:    "US",
			Categories: []string{"documentary"},
		},
	}, []iptvLogo{

		{

			Channel: "ABC.us",
			InUse:   true,
			Width:   512,
			Height:  512,
			Format:  "PNG",
			URL:     "https://example.test/abc.png",
		},
		{

			Channel: "AmericanHeroesChannel.us",
			InUse:   true,
			Width:   512,
			Height:  288,
			Format:  "PNG",
			URL:     "https://example.test/ahc.png",
		},
	})

	catalog := &ChannelCatalog{

		Channels: []Channel{

			{

				DaddyID:  "766",
				Name:     "abc ny usa",
				Slug:     "abc-ny-usa",
				Category: "Entertainment",
			},
			{

				DaddyID:  "206",
				Name:     "ahc (american heroes channel)",
				Slug:     "ahc-american-heroes-channel",
				Category: "Entertainment",
			},
		},
	}

	count := enrichChannelCatalog(catalog, index)

	if count != 2 {

		t.Fatalf("expected 2 enriched channels, got %d", count)

	}

	if catalog.Channels[0].Name != "ABC" {

		t.Fatalf("expected regional ABC to become canonical ABC, got %q", catalog.Channels[0].Name)

	}

	if catalog.Channels[0].Country.Code != "us" || catalog.Channels[0].Category != "General" || catalog.Channels[0].Logo == "" {

		t.Fatalf("expected ABC country/category/logo enrichment, got %+v", catalog.Channels[0])

	}

	if catalog.Channels[1].Name != "American Heroes Channel" {

		t.Fatalf("expected AHC alias to match American Heroes Channel, got %q", catalog.Channels[1].Name)

	}

	if catalog.Channels[1].Category != "Documentary" || catalog.Channels[1].Logo == "" {

		t.Fatalf("expected AHC category/logo enrichment, got %+v", catalog.Channels[1])

	}

}
