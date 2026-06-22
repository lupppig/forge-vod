package ffmpeg

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
)

// EncodeOptions tunes a single rendition encode.
type EncodeOptions struct {
	SegmentDuration int    // target segment length in seconds (default 6)
	KeyInfoFile     string // path to an ffmpeg HLS key info file; when set, segments are AES-128 encrypted
}

// RenditionArgs builds the ffmpeg arguments to encode input into a single HLS
// rendition: an H.264/AAC variant playlist plus its TS segments under outDir.
// The variant playlist is named "<rendition>.m3u8" and segments
// "<rendition>_%03d.ts".
func RenditionArgs(input, outDir string, r Rendition, segmentDuration int) []string {
	return RenditionArgsWithOptions(input, outDir, r, EncodeOptions{SegmentDuration: segmentDuration})
}

// RenditionArgsWithOptions is RenditionArgs with full control over encode
// options, including optional AES-128 segment encryption via a key info file.
func RenditionArgsWithOptions(input, outDir string, r Rendition, opts EncodeOptions) []string {
	segmentDuration := opts.SegmentDuration
	if segmentDuration <= 0 {
		segmentDuration = 6
	}
	playlist := filepath.Join(outDir, r.Name+".m3u8")
	segmentPattern := filepath.Join(outDir, r.Name+"_%03d.ts")

	args := []string{
		"-y",
		"-i", input,
		"-vf", "scale=-2:" + strconv.Itoa(r.Height),
		"-pix_fmt", "yuv420p",
		"-c:v", "libx264",
		"-profile:v", "main",
		"-preset", "veryfast",
		"-b:v", strconv.Itoa(r.VideoKbps) + "k",
		"-maxrate", strconv.Itoa(r.VideoKbps*107/100) + "k",
		"-bufsize", strconv.Itoa(r.VideoKbps*15/10) + "k",
		"-c:a", "aac",
		"-b:a", strconv.Itoa(r.AudioKbps) + "k",
		"-ac", "2",
		"-hls_time", strconv.Itoa(segmentDuration),
		"-hls_playlist_type", "vod",
		"-hls_segment_filename", segmentPattern,
	}
	if opts.KeyInfoFile != "" {
		args = append(args, "-hls_key_info_file", opts.KeyInfoFile)
	}
	args = append(args, "-f", "hls", playlist)
	return args
}

// EncodeRendition runs ffmpeg to produce a single HLS rendition under outDir.
func EncodeRendition(ctx context.Context, run Runner, ffmpegPath, input, outDir string, r Rendition, segmentDuration int) error {
	return EncodeRenditionWithOptions(ctx, run, ffmpegPath, input, outDir, r, EncodeOptions{SegmentDuration: segmentDuration})
}

// EncodeRenditionWithOptions runs ffmpeg to produce a single HLS rendition under
// outDir using the supplied options.
func EncodeRenditionWithOptions(ctx context.Context, run Runner, ffmpegPath, input, outDir string, r Rendition, opts EncodeOptions) error {
	if _, err := run.Run(ctx, ffmpegPath, RenditionArgsWithOptions(input, outDir, r, opts)...); err != nil {
		return fmt.Errorf("encode %s: %w", r.Name, err)
	}
	return nil
}
