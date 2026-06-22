package ffmpeg

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
)

// RenditionArgs builds the ffmpeg arguments to encode input into a single HLS
// rendition: an H.264/AAC variant playlist plus its TS segments under outDir.
// The variant playlist is named "<rendition>.m3u8" and segments
// "<rendition>_%03d.ts".
func RenditionArgs(input, outDir string, r Rendition, segmentDuration int) []string {
	if segmentDuration <= 0 {
		segmentDuration = 6
	}
	playlist := filepath.Join(outDir, r.Name+".m3u8")
	segmentPattern := filepath.Join(outDir, r.Name+"_%03d.ts")

	return []string{
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
		"-f", "hls",
		playlist,
	}
}

// EncodeRendition runs ffmpeg to produce a single HLS rendition under outDir.
func EncodeRendition(ctx context.Context, run Runner, ffmpegPath, input, outDir string, r Rendition, segmentDuration int) error {
	if _, err := run.Run(ctx, ffmpegPath, RenditionArgs(input, outDir, r, segmentDuration)...); err != nil {
		return fmt.Errorf("encode %s: %w", r.Name, err)
	}
	return nil
}
