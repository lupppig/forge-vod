package ffmpeg

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFormatVTTTime(t *testing.T) {
	cases := map[float64]string{
		0:       "00:00:00.000",
		5.5:     "00:00:05.500",
		65:      "00:01:05.000",
		3661.25: "01:01:01.250",
	}
	for in, want := range cases {
		if got := formatVTTTime(in); got != want {
			t.Errorf("formatVTTTime(%v) = %q, want %q", in, got, want)
		}
	}
}

func TestBuildVTTCueCountAndGeometry(t *testing.T) {
	s := StoryboardSpec{Interval: 5, TileWidth: 160, TileHeight: 90, Cols: 5, Rows: 5}
	// 22s duration -> ceil(22/5) = 5 cues (at 0,5,10,15,20).
	vtt := BuildVTT("sprite.jpg", 22, s)

	if !strings.HasPrefix(vtt, "WEBVTT") {
		t.Fatal("VTT must start with WEBVTT header")
	}
	cues := strings.Count(vtt, "-->")
	if cues != 5 {
		t.Fatalf("expected 5 cues for 22s, got %d:\n%s", cues, vtt)
	}
	// Second tile (index 1) sits at x=160,y=0.
	if !strings.Contains(vtt, "sprite.jpg#xywh=160,0,160,90") {
		t.Errorf("missing expected second-tile geometry:\n%s", vtt)
	}
	// Sixth tile would wrap to row 2 (index 5 -> x=0,y=90) but duration cuts off first.
	if strings.Contains(vtt, "xywh=0,90") {
		t.Error("did not expect a second row given the short duration")
	}
}

func TestBuildVTTLastCueClampedToDuration(t *testing.T) {
	s := StoryboardSpec{Interval: 5, TileWidth: 160, TileHeight: 90, Cols: 5, Rows: 5}
	vtt := BuildVTT("s.jpg", 12, s) // cues at 0,5,10; last clamped to 12
	if !strings.Contains(vtt, "00:00:10.000 --> 00:00:12.000") {
		t.Errorf("expected last cue clamped to duration:\n%s", vtt)
	}
}

func TestStoryboardArgs(t *testing.T) {
	s := StoryboardSpec{Interval: 5, TileWidth: 160, TileHeight: 90, Cols: 5, Rows: 5}
	args := StoryboardArgs("in.mp4", "sprite.jpg", s)
	if !containsPair(args, "-vf", "fps=0.2,scale=160:90,tile=5x5") {
		t.Errorf("unexpected filtergraph in args: %v", args)
	}
	if !containsPair(args, "-frames:v", "1") {
		t.Error("expected single output frame")
	}
}

func TestGenerateStoryboardIntegration(t *testing.T) {
	dir := t.TempDir()
	src := makeTestVideo(t, dir)
	out := filepath.Join(dir, "sprite.jpg")

	// 2s clip, 1s interval, small grid.
	s := StoryboardSpec{Interval: 1, TileWidth: 80, TileHeight: 60, Cols: 2, Rows: 2}
	if err := GenerateStoryboard(context.Background(), NewExecRunner(), "ffmpeg", src, out, s); err != nil {
		t.Fatalf("GenerateStoryboard: %v", err)
	}
	fi, err := os.Stat(out)
	if err != nil || fi.Size() == 0 {
		t.Fatalf("sprite sheet not produced: err=%v", err)
	}
}
