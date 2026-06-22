package ffmpeg

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestThumbnailTimestamp(t *testing.T) {
	cases := []struct {
		duration float64
		want     float64
	}{
		{duration: 0, want: 0},
		{duration: 100, want: 10},
		{duration: 0.05, want: 0}, // very short: clamped, never negative
	}
	for _, c := range cases {
		if got := ThumbnailTimestamp(c.duration); got != c.want {
			t.Errorf("ThumbnailTimestamp(%v) = %v, want %v", c.duration, got, c.want)
		}
	}
}

func TestThumbnailArgs(t *testing.T) {
	args := ThumbnailArgs("in.mp4", "out.jpg", 5.5, 320)
	if !containsPair(args, "-ss", "5.500") {
		t.Error("missing/incorrect seek timestamp")
	}
	if !containsPair(args, "-vframes", "1") {
		t.Error("expected single frame capture")
	}
	if !containsPair(args, "-vf", "scale=320:-2") {
		t.Error("missing/incorrect scale filter")
	}
	if last := args[len(args)-1]; last != "out.jpg" {
		t.Errorf("expected output last, got %q", last)
	}
}

func TestGenerateThumbnailIntegration(t *testing.T) {
	dir := t.TempDir()
	src := makeTestVideo(t, dir)
	out := filepath.Join(dir, "thumb.jpg")

	if err := GenerateThumbnail(context.Background(), NewExecRunner(), "ffmpeg", src, out, 1.0, 160); err != nil {
		t.Fatalf("GenerateThumbnail: %v", err)
	}
	fi, err := os.Stat(out)
	if err != nil {
		t.Fatalf("thumbnail not produced: %v", err)
	}
	if fi.Size() == 0 {
		t.Fatal("thumbnail is empty")
	}
}
