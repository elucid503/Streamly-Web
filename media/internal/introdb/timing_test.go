package introdb

import "testing"

func TestIntroWindow_StartAtZero(t *testing.T) {
	end := int64(90_000)
	record := &MediaRecord{
		Intro: []SegmentTimestamp{{StartMs: 0, EndMs: &end}},
	}

	start, endMs, ok := IntroWindow(record)
	if !ok {
		t.Fatal("expected intro window")
	}
	if start != 0 {
		t.Fatalf("start = %d, want 0", start)
	}
	if endMs != 90_000 {
		t.Fatalf("end = %d, want 90000", endMs)
	}
}

func TestIntroSkipTarget_StartAtZero(t *testing.T) {
	end := int64(90_000)
	record := &MediaRecord{
		Intro: []SegmentTimestamp{{StartMs: 0, EndMs: &end}},
	}

	target, err := IntroSkipTarget(record, 15_000)
	if err != nil {
		t.Fatalf("IntroSkipTarget: %v", err)
	}
	if target.Milliseconds() != 90_000 {
		t.Fatalf("target = %d, want 90000", target.Milliseconds())
	}
}