package ffmpeg

import (
	"context"
	"os/exec"
	"sync"
	"testing"
)

// fakeRunner records calls and returns canned output, so stage logic can be
// tested without invoking real binaries.
type fakeRunner struct {
	mu     sync.Mutex
	calls  [][]string
	output []byte
	err    error
}

func (f *fakeRunner) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, append([]string{name}, args...))
	return f.output, f.err
}

// ffmpegAvailable reports whether real ffmpeg+ffprobe are on PATH, gating
// integration tests.
func ffmpegAvailable(t *testing.T) {
	t.Helper()
	for _, bin := range []string{"ffmpeg", "ffprobe"} {
		if _, err := exec.LookPath(bin); err != nil {
			t.Skipf("%s not available: %v", bin, err)
		}
	}
}

func TestParseProbe(t *testing.T) {
	raw := []byte(`{"streams":[{"width":1920,"height":1080}],"format":{"duration":"42.5"}}`)
	got, err := parseProbe(raw)
	if err != nil {
		t.Fatalf("parseProbe: %v", err)
	}
	if got.Width != 1920 || got.Height != 1080 || got.Duration != 42.5 {
		t.Fatalf("unexpected MediaInfo: %+v", got)
	}
}

func TestParseProbeNoStream(t *testing.T) {
	if _, err := parseProbe([]byte(`{"streams":[],"format":{}}`)); err == nil {
		t.Fatal("expected error for missing video stream")
	}
}

func TestLadderForCapsSource(t *testing.T) {
	got := LadderFor(720, DefaultLadder)
	if len(got) != 3 {
		t.Fatalf("expected 3 rungs for 720p source, got %d: %+v", len(got), got)
	}
	for _, r := range got {
		if r.Height > 720 {
			t.Fatalf("rung %s exceeds source height", r.Name)
		}
	}
}

func TestLadderForTinySource(t *testing.T) {
	got := LadderFor(240, DefaultLadder)
	if len(got) != 1 {
		t.Fatalf("expected single rung for tiny source, got %d", len(got))
	}
	if got[0].Height != 240 {
		t.Fatalf("expected rung matched to source height 240, got %d", got[0].Height)
	}
}

func TestRenditionArgs(t *testing.T) {
	r := Rendition{Name: "720p", Height: 720, VideoKbps: 2800, AudioKbps: 128}
	args := RenditionArgs("in.mp4", "/out", r, 6)

	if !containsPair(args, "-i", "in.mp4") {
		t.Error("missing input arg")
	}
	if !containsPair(args, "-vf", "scale=-2:720") {
		t.Error("missing/incorrect scale filter")
	}
	if !containsPair(args, "-hls_time", "6") {
		t.Error("missing hls_time")
	}
	if last := args[len(args)-1]; last != "/out/720p.m3u8" {
		t.Errorf("expected playlist output last, got %q", last)
	}
}

func TestProbeWithFakeRunner(t *testing.T) {
	f := &fakeRunner{output: []byte(`{"streams":[{"width":640,"height":360}],"format":{"duration":"10"}}`)}
	info, err := Probe(context.Background(), f, "ffprobe", "in.mp4")
	if err != nil {
		t.Fatalf("Probe: %v", err)
	}
	if info.Height != 360 {
		t.Fatalf("expected height 360, got %d", info.Height)
	}
	if len(f.calls) != 1 || f.calls[0][0] != "ffprobe" {
		t.Fatalf("expected one ffprobe call, got %+v", f.calls)
	}
}

// containsPair reports whether args contains flag immediately followed by val.
func containsPair(args []string, flag, val string) bool {
	for i := 0; i+1 < len(args); i++ {
		if args[i] == flag && args[i+1] == val {
			return true
		}
	}
	return false
}
