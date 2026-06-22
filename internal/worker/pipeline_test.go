package worker

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lupppig/forge-vod/internal/ffmpeg"
)

// fakeStore records download/upload calls without touching object storage.
type fakeStore struct {
	downloaded string
	uploaded   string
}

func (f *fakeStore) DownloadRaw(_ context.Context, objectKey, dst string) error {
	f.downloaded = objectKey
	return os.WriteFile(dst, []byte("fake source"), 0o644)
}

func (f *fakeStore) UploadHLSDir(_ context.Context, _, prefix string) error {
	f.uploaded = prefix
	return nil
}

// fakeEncoder records the order stages were invoked and writes placeholder
// outputs so later stages and uploads see files.
type fakeEncoder struct {
	steps  []string
	keyURI string
	keyDir string
}

func (e *fakeEncoder) Probe(context.Context, string) (ffmpeg.MediaInfo, error) {
	e.steps = append(e.steps, "probe")
	return ffmpeg.MediaInfo{Width: 1280, Height: 720, Duration: 30}, nil
}

func (e *fakeEncoder) GenerateKey() ([]byte, error) {
	e.steps = append(e.steps, "genkey")
	return make([]byte, 16), nil
}

func (e *fakeEncoder) WriteKeyMaterial(dir, keyURI string, _ []byte) (ffmpeg.KeyInfo, error) {
	e.steps = append(e.steps, "writekey")
	e.keyURI = keyURI
	e.keyDir = dir
	return ffmpeg.KeyInfo{KeyURI: keyURI, KeyPath: filepath.Join(dir, "enc.keyinfo")}, nil
}

func (e *fakeEncoder) EncodeRendition(_ context.Context, _, outDir string, r ffmpeg.Rendition, _ ffmpeg.EncodeOptions) error {
	e.steps = append(e.steps, "encode:"+r.Name)
	return os.WriteFile(filepath.Join(outDir, r.Name+".m3u8"), []byte("#EXTM3U"), 0o644)
}

func (e *fakeEncoder) GenerateThumbnail(_ context.Context, _, out string, _ float64, _ int) error {
	e.steps = append(e.steps, "thumbnail")
	return os.WriteFile(out, []byte("jpg"), 0o644)
}

func (e *fakeEncoder) GenerateStoryboard(_ context.Context, _, out string, _ ffmpeg.StoryboardSpec) error {
	e.steps = append(e.steps, "storyboard")
	return os.WriteFile(out, []byte("jpg"), 0o644)
}

func (e *fakeEncoder) WriteFile(name string, data []byte) error {
	e.steps = append(e.steps, "write:"+filepath.Base(name))
	return os.WriteFile(name, data, 0o644)
}

func TestPipelineRunsAllStagesInOrder(t *testing.T) {
	store := &fakeStore{}
	enc := &fakeEncoder{}
	p := &Pipeline{
		Store:      store,
		Encoder:    enc,
		Log:        slog.New(slog.NewTextHandler(io.Discard, nil)),
		KeyURIBase: "https://api.example.com",
	}

	workDir := t.TempDir()
	res, err := p.Process(context.Background(), "vid-1", "vid-1.mp4", workDir, "vid-1")
	if err != nil {
		t.Fatalf("Process: %v", err)
	}

	// 720p source -> 720p, 480p, 360p (1080p dropped, no upscaling).
	if len(res.Renditions) != 3 {
		t.Fatalf("expected 3 renditions for 720p source, got %d", len(res.Renditions))
	}

	joined := strings.Join(enc.steps, ",")
	// Probe must precede encoding; key material before encoding; manifest last write.
	if !strings.HasPrefix(joined, "probe,genkey,writekey,encode:720p") {
		t.Fatalf("unexpected stage order: %s", joined)
	}
	for _, want := range []string{"thumbnail", "storyboard", "write:storyboard.vtt", "write:master.m3u8"} {
		if !strings.Contains(joined, want) {
			t.Errorf("missing stage %q in: %s", want, joined)
		}
	}
	// Thumbnail and storyboard must come after all encodes.
	if strings.Index(joined, "thumbnail") < strings.LastIndex(joined, "encode:") {
		t.Error("thumbnail should run after renditions")
	}

	if store.downloaded != "vid-1.mp4" {
		t.Errorf("expected source download, got %q", store.downloaded)
	}
	if store.uploaded != "vid-1" {
		t.Errorf("expected upload under prefix vid-1, got %q", store.uploaded)
	}
	if res.MasterKey != "vid-1/master.m3u8" {
		t.Errorf("unexpected master key %q", res.MasterKey)
	}

	// The playlist key URI must point at the per-video API key endpoint.
	if want := "https://api.example.com/videos/vid-1/key"; enc.keyURI != want {
		t.Errorf("key URI = %q, want %q", enc.keyURI, want)
	}
	// The key material must be written OUTSIDE the uploaded output dir so it is
	// never published to the public bucket.
	uploadDir := filepath.Join(workDir, "out")
	if strings.HasPrefix(enc.keyDir, uploadDir) {
		t.Errorf("key dir %q is inside uploaded dir %q; key would leak", enc.keyDir, uploadDir)
	}
	// The AES key is returned for persistence, not uploaded (16 bytes -> 32 hex).
	if len(res.EncKeyHex) != 32 {
		t.Errorf("expected 32-char hex key, got %q", res.EncKeyHex)
	}
}
