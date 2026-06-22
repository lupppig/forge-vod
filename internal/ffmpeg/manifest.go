package ffmpeg

import (
	"fmt"
	"strings"
)

// BuildMasterPlaylist renders an HLS master playlist (#EXTM3U) referencing each
// rendition's variant playlist ("<name>.m3u8"). Renditions are emitted in the
// order given; pass them highest-quality first if you want players to see the
// top rung first. Each variant advertises BANDWIDTH and RESOLUTION so adaptive
// players can choose a rung.
func BuildMasterPlaylist(renditions []Rendition) string {
	var b strings.Builder
	b.WriteString("#EXTM3U\n")
	b.WriteString("#EXT-X-VERSION:3\n")

	for _, r := range renditions {
		width := r.Height * 16 / 9
		// Ensure even width, matching the scale=-2 behaviour of the encoder.
		if width%2 != 0 {
			width++
		}
		fmt.Fprintf(&b, "#EXT-X-STREAM-INF:BANDWIDTH=%d,RESOLUTION=%dx%d\n", r.Bandwidth, width, r.Height)
		fmt.Fprintf(&b, "%s.m3u8\n", r.Name)
	}
	return b.String()
}
