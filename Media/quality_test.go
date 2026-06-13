package mediakit

import "testing"

func TestIsHLSURL(t *testing.T) {
	cases := []struct {
		url  string
		want bool
	}{
		{"https://cdn.example.com/video.mp4", false},
		{"https://cdn.example.com/playlist.m3u8", true},
		{"https://dami-tv.pro/papi/tv/playlist/abc", true},
	}

	for _, tc := range cases {
		if got := IsHLSURL(tc.url); got != tc.want {
			t.Errorf("IsHLSURL(%q) = %v, want %v", tc.url, got, tc.want)
		}
	}
}

func TestPickQualityPrefersProgressive(t *testing.T) {
	qualities := []Quality{
		{URL: "https://cdn.example.com/hd.m3u8", Height: 1080, IsHLS: true},
		{URL: "https://cdn.example.com/hd.mp4", Height: 720, IsHLS: false},
	}

	picked := PickQuality(qualities, 1080)
	if picked == nil || picked.IsHLS {
		t.Fatalf("expected progressive quality, got %+v", picked)
	}
}

func TestPickQualityCapsAtTarget(t *testing.T) {
	qualities := []Quality{
		{URL: "https://cdn.example.com/720.mp4", Height: 720, IsHLS: false},
		{URL: "https://cdn.example.com/2160.mp4", Height: 2160, IsHLS: false},
	}

	picked := PickQuality(qualities, 1080)
	if picked == nil || picked.Height != 720 {
		t.Fatalf("expected 720p, got %+v", picked)
	}
}

func TestPickNextLowerQuality(t *testing.T) {
	qualities := []Quality{
		{URL: "https://cdn.example.com/720.mp4", Height: 720, IsHLS: false},
		{URL: "https://cdn.example.com/1080.mp4", Height: 1080, IsHLS: false},
	}

	picked := PickNextLowerQuality(qualities, 1080)
	if picked == nil || picked.Height != 720 {
		t.Fatalf("expected 720p, got %+v", picked)
	}

	if PickNextLowerQuality(qualities, 720) != nil {
		t.Fatal("expected no lower quality below 720p")
	}
}

func TestEpisodeNumbers(t *testing.T) {
	season, episode := episodeNumbers("Breaking.Bad.S02E05.720p.mkv")
	if season != 2 || episode != 5 {
		t.Fatalf("got season=%d episode=%d", season, episode)
	}

	_, episode = episodeNumbers("Show.Episode.12.mp4")
	if episode != 12 {
		t.Fatalf("got episode=%d", episode)
	}
}