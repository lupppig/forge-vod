package ffmpeg

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// makeTestVideo synthesizes a short 320x240 test clip with audio using ffmpeg's
// lavfi sources, returning its path. Skips the test if ffmpeg is unavailable.
func makeTestVideo(t *testing.T, dir string) string {
	t.Helper()
	ffmpegAvailable(t)
	path := filepath.Join(dir, "src.mp4")
	run := NewExecRunner()
	args := []string{
		"-y",
		"-f", "lavfi", "-i", "testsrc=duration=2:size=320x240:rate=15",
		"-f", "lavfi", "-i", "sine=frequency=440:duration=2",
		"-c:v", "libx264", "-c:a", "aac", "-shortest",
		path,
	}
	if _, err := run.Run(context.Background(), run.FFmpegPath, args...); err != nil {
		t.Fatalf("create test video: %v", err)
	}
	return path
}

func TestProbeIntegration(t *testing.T) {
	dir := t.TempDir()
	src := makeTestVideo(t, dir)

	info, err := Probe(context.Background(), NewExecRunner(), "ffprobe", src)
	if err != nil {
		t.Fatalf("Probe: %v", err)
	}
	if info.Width != 320 || info.Height != 240 {
		t.Fatalf("expected 320x240, got %dx%d", info.Width, info.Height)
	}
	if info.Duration < 1.5 || info.Duration > 2.5 {
		t.Fatalf("expected ~2s duration, got %f", info.Duration)
	}
}

func TestEncodeRenditionIntegration(t *testing.T) {
	dir := t.TempDir()
	src := makeTestVideo(t, dir)
	outDir := filepath.Join(dir, "hls")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatal(err)
	}

	r := Rendition{Name: "240p", Height: 240, VideoKbps: 400, AudioKbps: 64}
	if err := EncodeRendition(context.Background(), NewExecRunner(), "ffmpeg", src, outDir, r, 2); err != nil {
		t.Fatalf("EncodeRendition: %v", err)
	}

	if _, err := os.Stat(filepath.Join(outDir, "240p.m3u8")); err != nil {
		t.Fatalf("variant playlist not produced: %v", err)
	}
	segments, _ := filepath.Glob(filepath.Join(outDir, "240p_*.ts"))
	if len(segments) == 0 {
		t.Fatal("no .ts segments produced")
	}
}
