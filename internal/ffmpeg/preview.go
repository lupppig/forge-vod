package ffmpeg

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
)

// StoryboardSpec describes a storyboard sprite sheet: one frame sampled every
// Interval seconds, scaled to TileWidth x TileHeight, tiled into a Cols x Rows
// grid image.
type StoryboardSpec struct {
	Interval   float64 // seconds between sampled frames
	TileWidth  int
	TileHeight int
	Cols       int
	Rows       int
}

// DefaultStoryboard returns sensible defaults for a 5s interval, 160x90 tiles in
// a 5x5 grid (25 tiles).
func DefaultStoryboard() StoryboardSpec {
	return StoryboardSpec{Interval: 5, TileWidth: 160, TileHeight: 90, Cols: 5, Rows: 5}
}

// TileCount is the maximum number of tiles the grid can hold.
func (s StoryboardSpec) TileCount() int { return s.Cols * s.Rows }

// StoryboardArgs builds ffmpeg args producing a single sprite-sheet image at out
// by sampling one frame per Interval seconds, scaling each, and tiling them.
func StoryboardArgs(input, out string, s StoryboardSpec) []string {
	fps := 1.0 / s.Interval
	vf := fmt.Sprintf("fps=%s,scale=%d:%d,tile=%dx%d",
		strconv.FormatFloat(fps, 'f', -1, 64),
		s.TileWidth, s.TileHeight, s.Cols, s.Rows)
	return []string{
		"-y",
		"-i", input,
		"-frames:v", "1",
		"-vf", vf,
		"-q:v", "4",
		out,
	}
}

// formatVTTTime renders seconds as a WebVTT timestamp (HH:MM:SS.mmm).
func formatVTTTime(seconds float64) string {
	if seconds < 0 {
		seconds = 0
	}
	total := int(seconds)
	ms := int((seconds - float64(total)) * 1000)
	h := total / 3600
	m := (total % 3600) / 60
	s := total % 60
	return fmt.Sprintf("%02d:%02d:%02d.%03d", h, m, s, ms)
}

// BuildVTT produces a WebVTT file body mapping time ranges to regions of the
// sprite sheet (#xywh fragments). duration bounds how many tiles are emitted;
// spriteName is the sprite image filename referenced by each cue.
func BuildVTT(spriteName string, duration float64, s StoryboardSpec) string {
	var b strings.Builder
	b.WriteString("WEBVTT\n\n")

	for i := range s.TileCount() {
		start := float64(i) * s.Interval
		if start >= duration {
			break
		}
		end := start + s.Interval
		if end > duration {
			end = duration
		}
		col := i % s.Cols
		row := i / s.Cols
		x := col * s.TileWidth
		y := row * s.TileHeight

		b.WriteString(formatVTTTime(start))
		b.WriteString(" --> ")
		b.WriteString(formatVTTTime(end))
		b.WriteString("\n")
		fmt.Fprintf(&b, "%s#xywh=%d,%d,%d,%d\n\n", spriteName, x, y, s.TileWidth, s.TileHeight)
	}
	return b.String()
}

// GenerateStoryboard runs ffmpeg to produce the sprite sheet at out.
func GenerateStoryboard(ctx context.Context, run Runner, ffmpegPath, input, out string, s StoryboardSpec) error {
	if _, err := run.Run(ctx, ffmpegPath, StoryboardArgs(input, out, s)...); err != nil {
		return fmt.Errorf("storyboard: %w", err)
	}
	return nil
}

// SpriteName returns the base sprite filename used in a VTT cue for a path.
func SpriteName(spritePath string) string { return filepath.Base(spritePath) }
