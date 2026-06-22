package ffmpeg

// Rendition is a single quality rung in the HLS bitrate ladder.
type Rendition struct {
	Name      string // e.g. "720p"
	Height    int    // target height in pixels (width derives from aspect)
	VideoKbps int    // video bitrate
	AudioKbps int    // audio bitrate
	Bandwidth int    // advertised BANDWIDTH for the master playlist (bits/s)
}

// DefaultLadder is the standard quality ladder, highest first.
var DefaultLadder = []Rendition{
	{Name: "1080p", Height: 1080, VideoKbps: 5000, AudioKbps: 192, Bandwidth: 5500000},
	{Name: "720p", Height: 720, VideoKbps: 2800, AudioKbps: 128, Bandwidth: 3000000},
	{Name: "480p", Height: 480, VideoKbps: 1400, AudioKbps: 128, Bandwidth: 1600000},
	{Name: "360p", Height: 360, VideoKbps: 800, AudioKbps: 96, Bandwidth: 900000},
}

// LadderFor returns the renditions to encode for a source of the given height,
// dropping any rung taller than the source so we never upscale. When the source
// is shorter than the smallest rung, a single rung matching the source height is
// returned so there is always at least one rendition.
func LadderFor(sourceHeight int, ladder []Rendition) []Rendition {
	var out []Rendition
	for _, r := range ladder {
		if r.Height <= sourceHeight {
			out = append(out, r)
		}
	}
	if len(out) == 0 && len(ladder) > 0 {
		smallest := ladder[len(ladder)-1]
		smallest.Height = sourceHeight
		out = append(out, smallest)
	}
	return out
}
