package ffmpeg

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
)

// MediaInfo describes the source video properties needed to plan encoding.
type MediaInfo struct {
	Width    int
	Height   int
	Duration float64 // seconds
}

// ProbeArgs builds the ffprobe arguments to emit JSON for the first video
// stream plus container format.
func ProbeArgs(input string) []string {
	return []string{
		"-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "stream=width,height:format=duration",
		"-of", "json",
		input,
	}
}

type ffprobeOutput struct {
	Streams []struct {
		Width  int `json:"width"`
		Height int `json:"height"`
	} `json:"streams"`
	Format struct {
		Duration string `json:"duration"`
	} `json:"format"`
}

// parseProbe converts raw ffprobe JSON into MediaInfo.
func parseProbe(raw []byte) (MediaInfo, error) {
	var out ffprobeOutput
	if err := json.Unmarshal(raw, &out); err != nil {
		return MediaInfo{}, fmt.Errorf("ffprobe: parse json: %w", err)
	}
	if len(out.Streams) == 0 {
		return MediaInfo{}, fmt.Errorf("ffprobe: no video stream found")
	}
	info := MediaInfo{
		Width:  out.Streams[0].Width,
		Height: out.Streams[0].Height,
	}
	if out.Format.Duration != "" {
		d, err := strconv.ParseFloat(out.Format.Duration, 64)
		if err != nil {
			return MediaInfo{}, fmt.Errorf("ffprobe: parse duration: %w", err)
		}
		info.Duration = d
	}
	return info, nil
}

// Probe runs ffprobe against input and returns its MediaInfo.
func Probe(ctx context.Context, r Runner, ffprobePath, input string) (MediaInfo, error) {
	raw, err := r.Run(ctx, ffprobePath, ProbeArgs(input)...)
	if err != nil {
		return MediaInfo{}, err
	}
	return parseProbe(raw)
}
