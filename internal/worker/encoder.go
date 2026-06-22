package worker

import (
	"context"
	"os"

	"github.com/lupppig/forge-vod/internal/ffmpeg"
)

// FFmpegEncoder is the production Encoder backed by the ffmpeg package and the
// real binaries on PATH.
type FFmpegEncoder struct {
	runner      ffmpeg.Runner
	ffmpegPath  string
	ffprobePath string
}

// NewFFmpegEncoder returns an Encoder using ffmpeg/ffprobe from PATH.
func NewFFmpegEncoder() *FFmpegEncoder {
	r := ffmpeg.NewExecRunner()
	return &FFmpegEncoder{runner: r, ffmpegPath: r.FFmpegPath, ffprobePath: r.FFprobePath}
}

func (e *FFmpegEncoder) Probe(ctx context.Context, input string) (ffmpeg.MediaInfo, error) {
	return ffmpeg.Probe(ctx, e.runner, e.ffprobePath, input)
}

func (e *FFmpegEncoder) GenerateKey() ([]byte, error) { return ffmpeg.GenerateKey() }

func (e *FFmpegEncoder) WriteKeyMaterial(dir, keyURI string, key []byte) (ffmpeg.KeyInfo, error) {
	return ffmpeg.WriteKeyMaterial(dir, keyURI, key)
}

func (e *FFmpegEncoder) EncodeRendition(ctx context.Context, input, outDir string, r ffmpeg.Rendition, opts ffmpeg.EncodeOptions) error {
	return ffmpeg.EncodeRenditionWithOptions(ctx, e.runner, e.ffmpegPath, input, outDir, r, opts)
}

func (e *FFmpegEncoder) GenerateThumbnail(ctx context.Context, input, out string, at float64, width int) error {
	return ffmpeg.GenerateThumbnail(ctx, e.runner, e.ffmpegPath, input, out, at, width)
}

func (e *FFmpegEncoder) GenerateStoryboard(ctx context.Context, input, out string, s ffmpeg.StoryboardSpec) error {
	return ffmpeg.GenerateStoryboard(ctx, e.runner, e.ffmpegPath, input, out, s)
}

func (e *FFmpegEncoder) WriteFile(name string, data []byte) error {
	return os.WriteFile(name, data, 0o644)
}
