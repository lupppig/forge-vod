package ffmpeg

import (
	"context"
	"fmt"
	"strconv"
)

// ThumbnailArgs builds the ffmpeg arguments to capture a single poster frame
// from input at atSeconds, scaled to width (height auto, aspect preserved),
// written as a JPEG to out.
func ThumbnailArgs(input, out string, atSeconds float64, width int) []string {
	if width <= 0 {
		width = 640
	}
	return []string{
		"-y",
		"-ss", strconv.FormatFloat(atSeconds, 'f', 3, 64),
		"-i", input,
		"-vframes", "1",
		"-vf", "scale=" + strconv.Itoa(width) + ":-2",
		"-q:v", "3",
		out,
	}
}

// ThumbnailTimestamp picks a sensible capture time: 10% into the video, capped
// so we never seek past a very short clip.
func ThumbnailTimestamp(duration float64) float64 {
	if duration <= 0 {
		return 0
	}
	at := duration * 0.1
	if max := duration - 0.1; at > max {
		at = max
	}
	if at < 0 {
		at = 0
	}
	return at
}

// GenerateThumbnail captures a poster frame from input at atSeconds into out.
func GenerateThumbnail(ctx context.Context, run Runner, ffmpegPath, input, out string, atSeconds float64, width int) error {
	if _, err := run.Run(ctx, ffmpegPath, ThumbnailArgs(input, out, atSeconds, width)...); err != nil {
		return fmt.Errorf("thumbnail: %w", err)
	}
	return nil
}
